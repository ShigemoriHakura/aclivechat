# aclivechat
用于OBS/XSplit的仿YouTube风格的AcFun直播评论栏/弹幕姬

![XSplit截图](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/xsplit.png)  
![OBS截图](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/obs.png)  
![OBS截图2](https://raw.githubusercontent.com/ShigemoriHakura/aclivechat/master/screenshots/obs2.jpg)  

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

### 置顶时间表
* 1   * 计算汇率 元 - 0 分钟  （蓝）
* 2   * 计算汇率 元 - 0 分钟  （浅蓝）
* 5   * 计算汇率 元 - 2 分钟  （绿）
* 10  * 计算汇率 元 - 5 分钟  （黄）
* 20  * 计算汇率 元 - 10 分钟 （橙）
* 50  * 计算汇率 元 - 30 分钟 （品红）
* 100 * 计算汇率 元 - 60 分钟 （红）

### 更新日志
## Frontend
**0.1.11**
* 修复小数问题（js挨打）

**0.1.10**
* 加入前端自定义文本

**0.1.9**
* 修复房间号传输错误

**0.1.8**
* 加入粉丝牌显示
* 加入用户标记显示

**0.1.7**
* 加入AC币代替实际价格的功能

**0.1.6**
* 修复SC固定后特定消息导致上下错误移动的问题

**0.1.5**
* 加入自定义计算汇率

**0.1.4**
* 更新价格为人民币计算标准

**0.1.3**
* 修改时间显示方式

**0.1.2**
* 修复退出拼写错误

**0.1.1**
* 更改了帮助图片

**0.1.0**
* 完善显示部分以及可自定义内容
* 移除大部分B显示

## Backend
**0.1.6**
* 加入房管判断

**0.1.5**
* 分离不同功能到不同文件
* 完善错误处理
* 加入登录机制

**0.1.4**
* 完善用户标记功能
* 加入粉丝牌传输

**0.1.3**
* 对不起我价格计算少打一个0

**0.1.2**
* 修复重复监听未关闭的bug
* 完善日志显示

**0.1.1**
* 修改panicln
* 完善部分提示内容

**0.1.0**
* 加入离开提示（前50个观众）

**0.0.12**
* 加入自定义文本至配置文件
* 同步acfundanmu版本
* 加入获取头像失败时使用默认头像
