# 规范目的
为了规范 git commit ，方便追踪每次提交的功能，制定以下 git commit 规范。

# commit 规范
commit 以 `type(module): message` 为统一规范，其中 type、module、message 均为每次提交的实际值，每一个字段的解释如下：

## type
* feat：新功能（feature）
* fix：修补 bug
* docs：更新文档（documentation）
* style： 格式变动（不影响代码运行的变动）
* refactor：重构（即不是新增功能，也不是修改bug的代码变动）
* test：增加测试 case
* chore：构建过程或辅助工具的变动

## module
指的是更新的模块，当前的值为：
* (nocalhost-)api
* nhctl
* (nocalhost-)dep
* (nocalhost-)web

括号内可不填。

## message
message 是本次实际的提交信息，统一使用 `英文` 提交。

# 例子
例如，本次提交新增了 nocalhost-api 的 userInfo 接口，那么 commit 可能是：
```
$ git commit -a -m "feat(api): add userInfo api"
```