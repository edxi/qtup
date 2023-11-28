# QT影像批量上传程序

```bash
Usage of qtup:
  -d    delete-pac参数的短写
  -delete-pac
        成功发送到Modality后删除PAC上的IMAGES。只有当设置了modality-send参数时才有效，默认值不删除。 
         对应环境变量DELETE-PAC
  -l string
        logfile参数的短写
  -logfile string
        日志文件，缺省不填写即输出到控制台。 
         对应环境变量LOGFILE
  -m string
        modality-send参数的短写
  -modality-send string
        上传后完成后，发送到的Modality名字。设置该参数上传后会发送，缺省不填写即不发送。 
         对应环境变量MODALITY-SEND
  -p string
        path-images参数的短写
  -path-images string
        需要上传的影像目录，默认为当前目录。 
         对应环境变量PATH-IMAGES
  -s string
        server参数的短写
  -server string
        影像上传的目标服务器，例如：https://hospital-pacs.beta.cn.xijiabrainmap.com 
         对应环境变量SERVER
  -t int (待开发)
        maxthread参数的短写 (default 3)
  -threads int （待开发）
        同时上传处理的线程数。这里线程不是同一个文件多线程，是目录下多个文件同时上传。默认值3。 
         对应环境变量THREADS (default 3)
```

例子：

```bash
# 上传 当前目录 的影像到 https://pac.qtserver.local服务器
qtup https://pac.qtserver.local
```


```bash
# 带所有参数
# 上传 local-image-path目录下 的影像到 https://pac.qtserver.local服务器，最多同时5个影像一起上传，
# 上传到服务器的影像发送到gapserver，发送后删除pac.qtserver.local服务器上的影像
# 输出写入日志文件20231127.log
qtup --server https://pac.qtserver.local --threads 5 --path-images local-image-path --modality-send gapserver --delete-pac true --logfile=20231127.log 
```


```bash
# 使用环境变量
# 上传./images目录下的影像到https://pac.qtserver.local服务器，输出写入日志文件20231127.log
export SERVER=https://pac.qtserver.local
export PATH-IMAGES=./images
export LOGFILE=./log/20231127.log
qtup
```

```bash
# 使用.env文件保存环境变量
# 上传 当前目录 的影像到 https://pac.qtserver.local服务器
cat > .env <<EOF
SERVER=https://pac.qtserver.local
EOF

qtup
```

* 上述参数用法可以混合使用，比如

  ```bash
  # 同时使用了环境变量、.env、参数（长短参数名、server可以省略参数名）
  export SERVER=https://pac-dev.qtserver.local
  cat > .env <<EOF
  SERVER=https://pac-test.qtserver.local
  PATH-IMAGES=Downloads/images
  EOF
  qtup https://pac.qtserver.local -t 1	 --logfile=./logs/20231127.log
  ```
* 参数没有顺序
* server参数值或环境变量必填，server参数可以不写参数名
* 优先级：参数>环境变量>.env文件
* 任何不填写的参数，会检查：

  * 是否有环境变量，有就使用
  * 没有对应环境变量就检查.env文件，有就使用
  * 都没有就检查该参数可缺省，有就使用默认值，没有就报错
