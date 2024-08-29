package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// 获取目录中的 .zip 文件列表
func getZipFiles(path string) ([]string, error) {
	var files []string

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".zip") {
			files = append(files, filePath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

func apiGet(server string) {
	resp, err := http.Get(server)
	if err != nil {
		log.Fatalln("请求失败: ", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalln("服务器返回错误状态码:", resp.StatusCode)
	}
	var data json.RawMessage
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Fatal("解析响应失败:", err)
	}
}

func uploadFile(server string, filePath string) (patientId string, err error) {
	// 读取 zip 文件内容
	zipData, err := os.ReadFile(filePath)
	if err != nil {
		log.Println("ERROR: 上传", zipData, "失败。 无法读取文件: ", err)
		return "", err
	}

	// 构建请求体
	body := bytes.NewReader(zipData)

	// 发送 POST 请求
	resp, err := http.Post(server, "application/zip", body)
	if err != nil {
		log.Println("ERROR: 上传", zipData, "失败。 POST 请求失败: ", err)
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		log.Println("ERROR: 上传", zipData, "失败。 服务器返回错误状态码: ", resp.StatusCode)
		return "", err
	}

	var data []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Println("ERROR: 上传", zipData, "失败。 解析响应失败:", err)
		return "", err
	}
	// 遍历切片查找第一个具有 "patientId" 字段的元素
	for _, item := range data {
		if val, ok := item["ParentPatient"].(string); ok {
			return val, nil
		}
	}

	log.Println("ERROR: 上传", zipData, "失败。 未找到具有 'patientId' ")
	return "", errors.New("未找到具有 'patientId' ")
}

func sentToModality(server string, patientId string) (jobId string, err error) {
	// 构建 JSON payload
	patientIds := []string{patientId}
	payload := struct {
		ID           []string `json:"Resources"`
		Asynchronous bool     `json:"Asynchronous"`
	}{
		ID:           patientIds,
		Asynchronous: true,
	}

	// 将 JSON 负载转换为字节数组
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Println("ERROR: patientId为", patientId, "在发送到modality时，payload转换失败:", err)
		return "", err
	}

	// 创建请求体
	body := bytes.NewReader(payloadBytes)

	// 发送 POST 请求
	resp, err := http.Post(server, "application/json", body)
	if err != nil {
		log.Println("ERROR: patientId为", patientId, "在发送到modality时，POST 请求失败:", err)
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		log.Println("ERROR: patientId为", patientId, "在发送到modality时，服务器返回错误状态码:", resp.StatusCode)
		return "", errors.New("在发送到modality时，服务器返回错误状态码")
	}
	// 解析响应
	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Println("ERROR: patientId为", patientId, "在发送到modality时，解析响应失败:", resp.StatusCode)
		return "", err
	}
	jobId, ok := data["ID"].(string)
	if !ok {
		log.Println("ERROR: patientId为", patientId, "在发送到modality时，找不到指定的ID键：")
		return "", errors.New("在发送到modality时，找不到指定的ID键")
	}

	return jobId, nil
}

func main() {
	// 定义命令行参数
	server := flag.String("server", "", "影像上传的目标服务器，例如：https://hospital-pacs-beta.cn.xijiabrainmap.com \n 对应环境变量SERVER")
	path := flag.String("path-images", "", "需要上传的影像目录，默认为当前目录。 \n 对应环境变量PATH-IMAGES")
	modality := flag.String("modality-send", "", "上传后完成后，发送到的Modality名字。设置该参数上传后会发送，缺省不填写即不发送。 \n 对应环境变量MODALITY-SEND")
	logfile := flag.String("logfile", "", "日志文件，缺省不填写即输出到控制台。 \n 对应环境变量LOGFILE")
	threads := flag.Int("threads", 3, "同时上传处理的线程数。这里线程不是同一个文件多线程，是目录下多个文件同时上传。默认值3。 \n 对应环境变量THREADS")
	deletePac := flag.Bool("delete-pac", false, "成功发送到Modality后删除PAC上的IMAGES。只有当设置了modality-send参数时才有效，默认值不删除。 \n 对应环境变量DELETE-PAC")
	flag.StringVar(server, "s", "", "server参数的短写")
	flag.StringVar(path, "p", "", "path-images参数的短写")
	flag.StringVar(modality, "m", "", "modality-send参数的短写")
	flag.StringVar(logfile, "l", "", "logfile参数的短写")
	flag.IntVar(threads, "t", 3, "maxthread参数的短写")
	flag.BoolVar(deletePac, "d", false, "delete-pac参数的短写")

	// 解析命令行参数
	flag.Parse()

	// 载入.env
	godotenv.Load(".env")

	//检查参数
	if *server == "" &&
		((func() bool { *server = flag.Arg(0); return *server == "" })()) &&
		((func() bool { *server = os.Getenv("SERVER"); return *server == "" })()) {
		flag.Usage()
		return
	} else {
		apiGet(*server + "/system")
	}

	if *modality != "" || ((func() bool { *modality = os.Getenv("MODALITY-SEND"); return *modality != "" })()) {
		apiGet(*server + "/modalities/" + *modality)
	}

	if *path == "" && ((func() bool { *path = os.Getenv("PATH-IMAGES"); return *path == "" })()) {
		*path = "./"
	}
	zipfiles, err := getZipFiles(*path)
	if err != nil {
		log.Fatalln("获取"+*path+"目录内.zip时发生错误:", err)
	}

	if *logfile != "" || ((func() bool { *logfile = os.Getenv("LOGFILE"); return *logfile != "" })()) {
		file, err := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalln("无法创建日志文件")
		}
		defer file.Close()

		log.SetOutput(file)
	}

	// 创建一个缓冲通道来限制并发的上传操作
	sem := make(chan struct{}, *threads)
	var wg sync.WaitGroup

	for _, zipfile := range zipfiles {
		wg.Add(1)
		go func(zipfile string) {
			defer wg.Done()
			sem <- struct{}{}        //获取一个信号量
			defer func() { <-sem }() //释放信号量

			// 处理单个文件的逻辑
			processFile(zipfile, *server, *modality, *deletePac)
		}(zipfile)
	}

	wg.Wait()
}

func processFile(zipfile, server, modality string, deletePac bool) {

	log.Println("INFO: ", zipfile, "开始上传。")
	patientId, err := uploadFile(server+"/instances", zipfile)
	if err != nil {
		return
	}
	log.Println("INFO: ", zipfile, "上传完成。对应patientId: ", patientId)
	if modality != "" {
		log.Println("INFO: 发送", zipfile, "对应的patientId为", patientId, "的影像到modality ", modality)
		jobId, err := sentToModality(server+"/modalities/"+modality+"/store", patientId)
		if err != nil {
			return
		}
		delayAttempts, maxDelayAttempts := 0, 40
		stateSuccess := false
		for ; delayAttempts < maxDelayAttempts; delayAttempts++ {
			resp, err := http.Get(server + "/jobs/" + jobId)
			if err != nil {
				log.Println("ERROR: 发送到modality的job:", jobId, " 状态检查请求失败: ", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Println("ERROR: 发送到modality的job:", jobId, " 服务器返回错误状态码:", resp.StatusCode)
			}
			var data map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&data)
			if err != nil {
				log.Println("ERROR: 发送到modality的job:", jobId, " 解析响应失败:", err)
			}
			jobState, ok := data["State"].(string)
			if !ok {
				log.Println("ERROR: 发送到modality的job:", jobId, " 找不到State键值")
			}
			if jobState == "Failure" {
				log.Println("ERROR: 发送到modality的job:", jobId, " State 报告失败")
				break
			}
			if jobState == "Success" {
				stateSuccess = true
				break
			}

			// 等待一段时间再发送下一次请求
			time.Sleep(60 * time.Second)
		}
		if !stateSuccess {
			return
		}
		log.Println("INFO: 成功发送", zipfile, "对应的patientId为", patientId, "的影像到modality ", modality)
		if deletePac {
			log.Println("INFO: 开始删除", zipfile, "对应的patientId为", patientId, "在pac server上的影像")
			req, err := http.NewRequest("DELETE", server+"/patients/"+patientId, nil)
			if err != nil {
				log.Println("ERROR: 删除patientId为", patientId, "在pac server上的影像时，创建 DELETE 请求失败: ", err)
				return
			}
			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				log.Println("ERROR: 删除patientId为", patientId, "在pac server上的影像时，发送 DELETE 请求失败: ", err)
				return
			}
			defer resp.Body.Close()

			// 检查响应状态码
			if resp.StatusCode != http.StatusOK {
				log.Println("ERROR: 删除patientId为", patientId, "在pac server上的影像时，服务器返回错误状态码: ", resp.StatusCode)
				return
			}

			log.Println("INFO: 成功删除", zipfile, "对应的patientId为", patientId, "在pac server上的影像")
		}
	}
}
