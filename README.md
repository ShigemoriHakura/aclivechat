# aclivechat
用于OBS的仿YouTube风格的ACFun直播评论栏

![XSplit截图](https://github.com/ShigemoriHakura/aclivechat/blob/master/screenshots/xsplit.png)  


## 感谢：
* 前端来自： https://github.com/xfgryujk/blivechat
* 后端弹幕获取： https://github.com/orzogc/acfundanmu

## 使用
* 运行aclivechat.exe
* 浏览器打开：[http://localhost:12451](http://localhost:12451)


**注意事项：**

* 应该先启动blivechat后启动OBS，否则网页会加载失败，这时应该刷新OBS的浏览器源，显示Loaded则加载成功
* 本地使用时不要关闭blivechat.exe那个黑框，否则不能继续获取弹幕
* 样式生成器没有列出所有本地字体，但是可以手动输入本地字体

### 源代码版
1. 编译前端（需要安装Node.js和npm）：
   ```sh
   cd frontend
   npm i
   npm run build
   ```
   
2. 编译后端（需要安装go）
   ```sh
   cd backend
   go get github.com/orzogc/acfundanmu
   go get github.com/gorilla/mux
   go get github.com/gorilla/websocket
   go get github.com/json-iterator/go
   go build
   ```
   
3. 正确放置文件
   ```sh
   前端 /frontend/dist
   后端 /
   ```

4. 用浏览器打开[http://localhost:12451](http://localhost:12451)，以下略