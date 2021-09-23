# fastflow
fastflow 是一个基于 DAG 执行任务流的框架，利用 go 的 goroutine 来执行任务，是以高性能、易拓展为目的而设计。

## 背景
组内有很多项目都涉及复杂的任务流场景，比如离线任务，集群上下架，容器迁移等等。以前体验过各种姿势去完成，比如airflow、在项目中手撸代码、借助第三方平台等等。
但是没有一个能全面满足我们的需求，有的功能上满足了，但性能太慢，有的性能高了，但可以复用性太差。于是我们着手自研了这样一个框架：
- 使用 DAG 的形式来定义任务流
- 基于协程的轻量级并发，而不是进程甚至是POD
- 支持水平扩容
- 支持多种存储可切换

## 特性
### 任务流
- [x] DAG执行(包含超时控制)
- [x] 重试任务
- [x] 取消任务
- [x] 模板参数
- [x] 任务间共享数据
- [ ] Cron定时调度
- [ ] 分布式锁

### 可扩展性
- [x] 水平扩容
- [x] 事件
- [ ] Hook

### 可观测性
可以通过exporter暴露以下
- [x] go status
- [x] process status任务
- [x] 任务执行总览
- [ ] 集群健康情况( master 与 心跳 )

### 多存储支持
配置存储:
- [x] mongo
- [ ] etcd
- [ ] redis
- [ ] mysql

数据存储:
- [x] mongo
- [ ] mysql

## Usage

参考 `examples` 目录

## 性能测试

MongoDB: 10core
MongoVersion: 4.2.5
FastflowInstances: 8
ExecutedTasksPerSecond: 2400