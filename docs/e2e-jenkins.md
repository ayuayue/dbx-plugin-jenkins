# 真实 Jenkins E2E

用临时 Jenkins 2.586（Java 21）验证 sidecar 的真实 REST 行为，不接触已有环境。

## 1. 准备临时 JENKINS_HOME

```text
jenkins-e2e/
  jenkins.war
    home/
    config.xml                 # 无安全域 + Unsecured 授权，含 all/Builds/Ops 三个视图
    jobs/demo/config.xml       # 打印 hello from dbx
    jobs/param-demo/config.xml # 字符串参数 BRANCH，打印 branch=%BRANCH%
    jobs/pipeline-demo/config.xml # Scripted Pipeline，7 个 stage
    jobs/slow-job/config.xml   # ping 45 秒，用于 stop 测试
```

`config.xml` 关键片段：

```xml
<securityRealm class="hudson.security.SecurityRealm$None"/>
<authorizationStrategy class="hudson.security.AuthorizationStrategy$Unsecured"/>
```

视图示例（`all` 为主视图，另加 `Builds`/`Ops`）：

```xml
<views>
  <hudson.model.AllView>
    <name>all</name>
  </hudson.model.AllView>
  <hudson.model.ListView>
    <name>Builds</name>
    <jobNames>
      <string>demo</string>
      <string>param-demo</string>
    </jobNames>
  </hudson.model.ListView>
  <hudson.model.ListView>
    <name>Ops</name>
    <jobNames>
      <string>slow-job</string>
    </jobNames>
  </hudson.model.ListView>
</views>
```

下载 WAR（腾讯云镜像，约 52MB）：

```powershell
curl.exe -L -o jenkins.war https://mirrors.cloud.tencent.com/jenkins/war/latest/jenkins.war
```

## 2. 启动

```powershell
$env:JENKINS_HOME = "C:\path\to\jenkins-e2e\home"
java -Djenkins.install.runSetupWizard=false `
     -Dhudson.model.DownloadService.noSignup=true `
     -Djenkins.model.Jenkins.slaveAgentPort=-1 `
     -jar jenkins.war --httpPort=18081 --prefix=/jenkins
```

> Git Bash 下必须加 `MSYS_NO_PATHCONV=1`，否则 `--prefix=/jenkins` 会被转成 Windows 路径，导致 `/jenkins` 404。

阶段视图需要安装插件（`workflow-aggregator`、`pipeline-rest-api`、`pipeline-model-definition`）：

```bash
curl -X POST -b cookies.txt -H "Jenkins-Crumb: $crumb" -H "Content-Type: text/xml" \
  --data '<jenkins><install plugin="workflow-aggregator@latest"/><install plugin="pipeline-rest-api@latest"/><install plugin="pipeline-model-definition@latest"/></jenkins>' \
  http://127.0.0.1:18081/jenkins/pluginManager/installNecessaryPlugins
```

安装完成后重启 Jenkins。

等 `http://127.0.0.1:18081/jenkins/api/json` 返回 200。

## 3. 运行 sidecar E2E

先编译 sidecar：

```powershell
$env:PATH = "<Go 安装目录>\bin;" + $env:PATH
cd backend
go build -trimpath -o ..\dist\dbx-plugin-jenkins.exe .
```

再执行：

```powershell
node scripts/e2e-jenkins.js dist\dbx-plugin-jenkins.exe
```

可用环境变量覆盖端点：

| 变量 | 默认 |
| --- | --- |
| `JENKINS_PORT` | `18081` |
| `JENKINS_BASE` | `/jenkins` |

## 4. 覆盖链路

- `connection/connect`：匿名 + `base_path=/jenkins` + CSRF 自动取 crumb
- `jenkins/whoami`：版本 `2.586`
- `jenkins/views`：`all` / `Builds` / `Ops`
- `jenkins/jobs` / `jenkins/job`：3 个任务、`BRANCH` 参数；`Builds` 视图只返回 demo/param-demo，`Ops` 只返回 slow-job
- `jenkins/trigger`：无参构建、带参构建
- `jenkins/builds` / `jenkins/build`：结果、耗时、触发原因
- `jenkins/console`：`progressiveText` 增量，包含 `hello from dbx` / `branch=dev`
- `jenkins/stop`：中止 `slow-job`，最终 `ABORTED`
- `jenkins/stages`：`pipeline-demo` 7 个 stage × 最近 5 次构建，`WorkflowJob` 类识别、平均耗时聚合
- `jenkins/stages` 变更集：需带 SCM 的 Pipeline。要点：装 git 插件并以 `-Dhudson.plugins.git.GitSCM.ALLOW_LOCAL_CHECKOUT=true` 启动；SCM URL 直接写盘符路径（`file:///C:/...` 会被 Git for Windows 改写成 `/C:/...`）；首次构建会 `Skipping changelog`，提交一次变更后再次构建才有数据。Pipeline 的 changelog 在 `changeSets`（复数），Freestyle 在 `changeSet`（单数），插件两者都读并按 commitId 去重
- `jenkins/queue` / `jenkins/nodes` / `jenkins/overview`
- `mcp/tools` / `mcp/call list_jobs`
- 只读连接触发构建 → `read_only: connection is in read-only mode`
- 视图 / 任务 CRUD：新建视图 → 校验任务归属 → 改描述与任务清单 → 新建自由风格任务 → 改描述/禁用 → 重命名 → 删除任务与视图 → 校验列表已清空
- 只读连接重复 CRUD（`createJob`、`deleteView`）→ `read_only`

## 5. 实测结果（2026-10-19）

```text
connect: {"message":"Connected to Jenkins 2.586","readOnly":false,...}
jobs: demo:success:false, param-demo:success:false, pipeline-demo:success:false, slow-job:aborted:false
views: Builds, Ops, all
Builds view: demo, param-demo
Ops view: slow-job
pipeline class: org.jenkinsci.plugins.workflow.job.WorkflowJob
stages: 获取构建人信息,参数验证,代码检出,打包构建,备份回滚应用,部署,启动应用 | builds 5 | avgRun 20961
stage build: {"number":5,"commits":0,"stages":7}
param-demo parameters: BRANCH=main
trigger demo: {"queueId":7,"success":true}
demo build: {"number":3,"result":"SUCCESS",...}
demo console: "...hello from dbx..."
param console: "...branch=dev..."
slow-job stopped: {"number":3,"result":"ABORTED",...}
nodes: Built-In Node:offline=false:0/2
overview: {"version":"2.586",...,"executorsBusy":0,"executorsTotal":2}
mcp tools: 10
read-only trigger: {"code":-32000,"message":"read_only: connection is in read-only mode"}
E2E OK
```

## 5.1 CRUD 实测（0.2.0）

```text
CRUD view jobs: demo
CRUD view updated: "updated by dbx" demo,param-demo
CRUD job updated: {"title":"crud-free","kind":"freestyle","description":"updated by dbx","disabled":true,"script":""}
CRUD job renamed: {"fullName":"crud-free-renamed","success":true}
CRUD pipeline skipped (workflow plugin unavailable): jenkins/createJob: server_error: Jenkins returned an HTML error page (HTTP 500); check the Jenkins server log
CRUD cleanup ok
read-only trigger: {"code":-32000,"message":"read_only: connection is in read-only mode"}
read-only createJob: {"code":-32000,"message":"read_only: connection is in read-only mode"}
read-only deleteView: {"code":-32000,"message":"read_only: connection is in read-only mode"}
E2E OK
```

- 视图创建使用 `POST /createView`，`name` 与 `mode` 同时出现在 form 和 `json` 中，任务清单一次性写入。
- 视图编辑走 `GET/POST /view/<name>/config.xml`，未修改字段（columns、comparator）保持不变。
- 任务编辑先 `doRename`（如有改名），再读取新路径的 config.xml 局部改写后写回。
- Pipeline 新建需实例安装 `workflow-job` / `workflow-cps`；本次实例插件下载超时未装，脚本自动跳过该段，Pipeline 模板由单测覆盖。
- 非 ListView（如 AllView）读取配置时 `fields` 返回 `null`，前端自动退化为原始 XML 编辑。

## 5.2 视图改名实测（0.4.0）

- 改名唯一入口：`POST /view/<old>/configSubmit`（`View.doConfigSubmit` → `rename()` → `owner.onViewRenamed`）；`POST config.xml` 的 `updateByXml` 明确拒绝覆盖名称。
- configSubmit 是**全量提交**：只带 `name` 时 `jobNames` 与 `columns` 会被清空、`recurse` 变 `true`（实测）。因此插件补齐 `json`（description/recurse/filterQueue/filterExecutors/columns）与每个任务的 `item_<job>`，有正则时补 `useincluderegex` / `includeRegex`。
- 改名后立刻用原 XML 回写新路径，恢复 `properties` / `jobFilters` 等表单覆盖不到的字段。
- 重名/非法名 Jenkins 返回 400 HTML 页面，插件在改名**前**用 `/api/json?tree=views[name]` 预校验（区分大小写）。

```text
CRUD view updated: "updated by dbx" demo,param-demo
CRUD view renamed: CRUD-View-Renamed "updated by dbx" demo,param-demo
CRUD cleanup ok
```

## 6. E2E 发现并修复的问题
**`computer.idle` 是布尔值，不是空闲执行器数量。**

原实现把 `idle` 当 int 解码，真实 Jenkins 返回 `"idle": true`，导致 `jenkins/nodes` 报
`cannot decode Jenkins response`。

修复：

- tree 增加 `executors[idle]`，按执行器明细统计空闲数；
- 旧版本无 `executors` 明细时，退化为 `idle` 布尔值（空闲数 = 1）；
- 单元测试 fixture 改为真实结构，并新增 `TestGetNodesFallbackIdleFlag` 回归用例。
