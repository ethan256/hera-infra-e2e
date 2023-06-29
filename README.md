# hera-infra-e2e是通过fork开源仓库skywalking-infra-e2e进行开发定制，其目的是为了帮助开发者设置、调试、验证E2E测试

## 概念和设计

### 项目目标

- 支持 `trace` 和 `metrics` 的端到端测试需求。
- 同时支持 `docker-compose` 和 `KinD` 来编排不同环境下的测试服务。
- 尽量做到语言无关，用户只需要配置 `YAML` 和运行命令，无需编写代码。
- 支持自动映射主机端口，以便实现并发E2E测试。

## 模块设计

### 控制器

控制器意味着组合配置文件中声明的所有步骤，它渐进并显示当前正在运行的步骤。如果它在某个步骤中失败，则可以显示错误消息，尽可能全面。例如：

```sh
e2e run
✔ Started Kind Cluster - Cluster Name
✔ Checked Pods Readiness - All pods are ready
? Generating Traffic - HTTP localhost:9090/users (progress spinner)
✔ Verified Output - service ls
(progress spinner) Verifying Output - endpoint ls
✘ Failed to Verify Output Data - endpoint ls
  <the diff content>
✔ Clean Up
```

### 测试步骤

Hera E2E测试可以分为以下几个步骤

**Setup**
启动本次E2E测试所需要的环境，如数据库、后端进程等

支持两种环境搭建方式：

- docker compose
  - 启动docker-compose服务。
  - 检查服务的健康状况。
  - 根据间隔等等到所有服务都准备好。
  - 执行命令设置测试环境或帮助验证，例如yq帮助评估 YAML 格式。
- KinD本地k8s集群
  - KinD根据配置文件启动集群。
  - 应用资源文件 ( --manifests) 或/并运行自定义初始化命令 ( --commands)。
  - 检查docker的准备情况。
  - 等到所有 pod 都根据interval等准备就绪。

**Trigger**
通过出发Action产生流量，可以按间隔访问HTTP API。支持两种触发方式：`http`和`hera-http`。其区别在于`hera-http`当触发成功时将不在触发，可以避免产生多条trace信息。

它可能有以下设置：

- interval：触发动作的频率。
- times：最大允许出发次数。0=infinite.  在某次触发成功后，不会再次触发。
- action：触发器的动作。

**Verify**
验证数据内容是否与预期结果匹配。比如单元测试断言等

它可能有以下设置：

- actual：实际的数据文件。
- query：获取实际数据的查询，可以运行 shell 命令来生成数据。
- expected：预期的数据文件，可以指定一些匹配规则来验证实际内容。

**Assert**
Verify中使用`go template`+[cpm](github.com/google/go-cmp/cmp)方式验证实际数据。在验证trace和metric时，存在以下问题：

- 在处理异步任务时，处理数组时，受数组中元素位置的影响，可能导致验证失败。例如，假设某个异步触发的trace中存在span1, span2, span3三个span，每次产生时span的顺序是不确定的，直接使用Verify方式将导致校验失败，其原因在于这里span的顺序无关。
- 无法跳过某些非重要的指标
- Assert目前仅支持特殊的数据格式，当trace为其他格式时，需要添加自定义数据。
- Assert采用sync.Pool方式减少GC，并通过并发提高校验速度。
- Assert支持metrics校验，并且允许忽略部分不重要的metrics。

Assert中存在一下设置：

- actual：实际的数据文件。
- query：获取metrics 数据的url。
- expected：预期的数据文件，可以指定一些匹配规则来验证实际内容。

**Cleanup**
此步骤需要设置步骤中的相同选项，以便它可以清理所有必要的东西。

- success：仅成功后清理
- failure：仅失败后清理
- always：不管任何结果，都清理
- never：永远不清理
