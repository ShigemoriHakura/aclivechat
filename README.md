# aclivechat
用于OBS/XSplit的仿YouTube风格的AcFun直播评论栏

![XSplit截图](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/xsplit.png)  
![OBS截图](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/obs.png)  

## 感谢：
* 前端来自： https://github.com/xfgryujk/blivechat
* 后端弹幕获取： https://github.com/orzogc/acfundanmu

## 使用
* 个性化修改config.json(可选)
* 运行aclivechat.exe
* 浏览器打开：[http://localhost:12451](http://localhost:12451)

**注意事项：**
* 每个前端版本更新后都需要重新复制一次链接和CSS至OBS/XSplit（因为有可能改动前端生成部分代码的）
* 应该先启动aclivechat后启动OBS/XSplit，否则网页会加载失败，失败时应刷新OBS/XSplit的浏览器源，显示Loaded则加载成功
* 本地使用时不能关闭aclivechat.exe，否则不能继续获取弹幕
* 样式生成器没有列出所有本地字体，可以手动输入本地字体
* 房间号就是UID
* 请关闭快速编辑模式（右键标题，选择属性，取消勾选快速编辑模式，然后确定）
![pin1](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/pin1.png)  
![pin2](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/pin2.png)  


### 直接下载
1. [Releases](https://github.com/ShigemoriHakura/aclivechat/releases)

### 从源代码编译
1. 编译前端（需要安装Node.js和npm【或者用cnpm更快】）：
   ```sh
   cd frontend
   npm i 【cnpm i】
   npm run build
   ```
   
2. 编译后端（需要安装go）
   ```sh
   go build
   ```
   
3. 正确放置文件
   ```sh
   前端 /dist
   后端 /
   ```

4. 浏览器打开[http://localhost:12451](http://localhost:12451)

### 功能列表
* 用户加入直播间显示
* 用户关注直播间显示
* 用户发送弹幕显示
* 用户点亮爱心显示
* 用户赠送礼物显示
* 自定义关注，加入，离开，点亮爱心文本


### 更新日志
## Frontend
**0.1.1**
* 更改了帮助图片

**0.1.0**
* 完善显示部分以及可自定义内容
* 移除大部分B显示

## Backend
**0.1.0**
* 加入离开提示（前50个观众）

**0.0.12**
* 加入自定义文本至配置文件
* 同步acfundanmu版本
* 加入获取头像失败时使用默认头像