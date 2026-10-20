import { spawn } from "node:child_process";

const exe = process.argv[2] || process.env.SIDECAR_EXE;
if (!exe) {
  console.error("usage: node scripts/e2e-jenkins.js <path-to-dbx-plugin-jenkins.exe>");
  process.exit(2);
}

const PORT = Number(process.env.JENKINS_PORT || 18081);
const BASE = process.env.JENKINS_BASE || "/jenkins";
const child = spawn(exe, [], { stdio: ["pipe", "pipe", "pipe"] });

let buffer = "";
const pending = new Map();

child.stdout.on("data", (data) => {
  buffer += data.toString("utf8");
  let index;
  while ((index = buffer.indexOf("\n")) >= 0) {
    const line = buffer.slice(0, index).trim();
    buffer = buffer.slice(index + 1);
    if (!line) continue;
    const message = JSON.parse(line);
    if (message.id && pending.has(message.id)) {
      pending.get(message.id)(message);
      pending.delete(message.id);
    }
  }
});

child.stderr.on("data", (data) => process.stderr.write(data));

function call(id, method, params, timeoutMs = 15000) {
  return new Promise((resolve, reject) => {
    pending.set(id, resolve);
    child.stdin.write(JSON.stringify({ jsonrpc: "2.0", id, method, params }) + "\n");
    setTimeout(() => {
      if (pending.has(id)) {
        pending.delete(id);
        reject(new Error(`timeout: ${method}`));
      }
    }, timeoutMs);
  });
}

let nextId = 100;
async function invoke(method, params, timeoutMs) {
  const response = await call(nextId++, method, params, timeoutMs);
  if (response.error) {
    throw new Error(`${method}: ${response.error.message}`);
  }
  return response.result;
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function waitFor(label, fn, timeoutMs = 60000) {
  const started = Date.now();
  let last;
  while (Date.now() - started < timeoutMs) {
    last = await fn();
    if (last) return last;
    await sleep(1000);
  }
  throw new Error(`timeout waiting for ${label}; last=${JSON.stringify(last)}`);
}

function connectionPayload(id, readOnly) {
  return {
    connection: {
      id,
      name: id,
      host: "127.0.0.1",
      port: PORT,
      username: "",
      external_config: { scheme: "http", base_path: BASE, read_only: readOnly },
    },
    runtime: { host: "127.0.0.1", port: PORT },
  };
}

async function readConsole(job, number) {
  let start = 0;
  let text = "";
  for (let i = 0; i < 30; i += 1) {
    const chunk = await invoke(
      "jenkins/console",
      { connectionId: "e2e", job, number, start },
      30000,
    );
    text += chunk.text;
    start = chunk.nextStart;
    if (!chunk.moreData) break;
    await sleep(500);
  }
  return text;
}

async function latestBuild(job) {
  const result = await invoke("jenkins/builds", { connectionId: "e2e", job, limit: 5 });
  return result.builds?.[0] ?? null;
}

(async () => {
  await call(1, "plugin/initialize", {
    host: {
      dbxVersion: "0.6.47",
      hostApiVersion: "1.4.0",
      features: [],
      protocolVersions: [1],
    },
    plugin: { id: "io.github.nixing178.jenkins", version: "0.3.0" },
    permissions: ["host.workbench"],
  });

  const connect = await invoke("connection/connect", connectionPayload("e2e", false));
  console.log("connect:", JSON.stringify(connect));

  const who = await invoke("jenkins/whoami", { connectionId: "e2e" });
  console.log("whoami:", JSON.stringify(who));

  const jobs = await invoke("jenkins/jobs", { connectionId: "e2e" });
  console.log(
    "jobs:",
    jobs.jobs.map((job) => `${job.name}:${job.status}:${job.isFolder}`).join(", "),
  );
  if (jobs.jobs.length < 3) throw new Error(`expected at least 3 jobs, got ${jobs.jobs.length}`);

  const views = await invoke("jenkins/views", { connectionId: "e2e" });
  console.log("views:", views.views.map((view) => view.name).join(", "));
  if (
    !views.views.some((view) => view.name === "Builds") ||
    !views.views.some((view) => view.name === "Ops")
  ) {
    throw new Error("expected Builds and Ops views");
  }

  const buildsView = await invoke("jenkins/jobs", { connectionId: "e2e", view: "Builds" });
  console.log("Builds view:", buildsView.jobs.map((job) => job.name).join(", "));
  if (buildsView.jobs.length !== 2 || buildsView.jobs.some((job) => job.name === "slow-job")) {
    throw new Error(`Builds view mismatch: ${JSON.stringify(buildsView.jobs.map((job) => job.name))}`);
  }

  const opsView = await invoke("jenkins/jobs", { connectionId: "e2e", view: "Ops" });
  console.log("Ops view:", opsView.jobs.map((job) => job.name).join(", "));
  if (opsView.jobs.length !== 1 || opsView.jobs[0].name !== "slow-job") {
    throw new Error(`Ops view mismatch: ${JSON.stringify(opsView.jobs.map((job) => job.name))}`);
  }

  const job = await invoke("jenkins/job", { connectionId: "e2e", job: "param-demo" });
  console.log(
    "param-demo parameters:",
    job.parameters.map((parameter) => `${parameter.name}=${parameter.defaultValue}`).join(","),
  );

  const beforeDemo = (await latestBuild("demo"))?.number ?? 0;
  const trigger = await invoke("jenkins/trigger", { connectionId: "e2e", job: "demo" });
  console.log("trigger demo:", JSON.stringify(trigger));

  const build = await waitFor("demo build to finish", async () => {
    const latest = await latestBuild("demo");
    return latest && latest.number > beforeDemo && latest.status === "success" ? latest : null;
  });
  console.log("demo build:", JSON.stringify(build));

  const consoleText = await readConsole("demo", build.number);
  if (!consoleText.includes("hello from dbx")) {
    throw new Error(`console missing marker: ${consoleText.slice(0, 200)}`);
  }
  console.log("demo console:", JSON.stringify(consoleText.trim()));

  const beforeParam = (await latestBuild("param-demo"))?.number ?? 0;
  const paramTrigger = await invoke("jenkins/trigger", {
    connectionId: "e2e",
    job: "param-demo",
    params: { BRANCH: "dev" },
  });
  console.log("trigger param-demo:", JSON.stringify(paramTrigger));

  const paramBuild = await waitFor("param-demo build to finish", async () => {
    const latest = await latestBuild("param-demo");
    return latest && latest.number > beforeParam && latest.status === "success" ? latest : null;
  });
  const paramConsole = await readConsole("param-demo", paramBuild.number);
  if (!paramConsole.includes("branch=dev")) {
    throw new Error(`parameter not applied: ${paramConsole.slice(0, 200)}`);
  }
  console.log("param console:", JSON.stringify(paramConsole.trim()));

  if (jobs.jobs.some((job) => job.name === "pipeline-demo")) {
    const pipelineDetail = await invoke("jenkins/job", {
      connectionId: "e2e",
      job: "pipeline-demo",
    });
    console.log("pipeline class:", pipelineDetail.class);
    if (!String(pipelineDetail.class ?? "").includes("WorkflowJob")) {
      throw new Error(`pipeline class not detected: ${pipelineDetail.class}`);
    }

    const beforePipeline = (await latestBuild("pipeline-demo"))?.number ?? 0;
    await invoke("jenkins/trigger", { connectionId: "e2e", job: "pipeline-demo" });
    const pipelineBuild = await waitFor(
      "pipeline-demo build to finish",
      async () => {
        const latest = await latestBuild("pipeline-demo");
        return latest && latest.number > beforePipeline && latest.status === "success"
          ? latest
          : null;
      },
      90000,
    );
    const matrix = await invoke("jenkins/stages", {
      connectionId: "e2e",
      job: "pipeline-demo",
      limit: 5,
    });
    console.log(
      "stages:",
      matrix.stageNames.join(","),
      "builds",
      matrix.builds.length,
      "avgRun",
      matrix.averageRun,
    );
    console.log("stage build:", JSON.stringify({
      number: matrix.builds[0].number,
      commits: matrix.builds[0].commitCount,
      stages: matrix.builds[0].stages?.length,
    }));
    if (matrix.stageNames.length < 7 || (matrix.builds[0].stages?.length ?? 0) < 7) {
      throw new Error(`stage matrix incomplete: ${JSON.stringify(matrix.stageNames)}`);
    }
    if (matrix.builds[0].number !== pipelineBuild.number) {
      throw new Error("stage matrix newest build mismatch");
    }
  }

  const beforeSlow = (await latestBuild("slow-job"))?.number ?? 0;
  await invoke("jenkins/trigger", { connectionId: "e2e", job: "slow-job" });
  const running = await waitFor("slow-job to start", async () => {
    const latest = await latestBuild("slow-job");
    return latest && latest.number > beforeSlow && latest.building ? latest : null;
  });
  console.log("slow-job running:", running.number);
  await invoke("jenkins/stop", { connectionId: "e2e", job: "slow-job", number: running.number });
  const stopped = await waitFor("slow-job to stop", async () => {
    const latest = await latestBuild("slow-job");
    return latest && !latest.building && latest.number === running.number ? latest : null;
  });
  console.log("slow-job stopped:", JSON.stringify(stopped));

  const queue = await invoke("jenkins/queue", { connectionId: "e2e" });
  console.log("queue:", queue.items.length);

  const nodes = await invoke("jenkins/nodes", { connectionId: "e2e" });
  console.log(
    "nodes:",
    nodes.nodes
      .map((node) => `${node.name}:offline=${node.offline}:${node.busy}/${node.numExecutors}`)
      .join(", "),
  );

  const overview = await invoke("jenkins/overview", { connectionId: "e2e" });
  console.log("overview:", JSON.stringify(overview));

  const tools = await invoke("mcp/tools", { connectionId: "e2e" });
  console.log("mcp tools:", tools.tools.length);
  const mcpCall = await invoke("mcp/call", {
    connectionId: "e2e",
    tool: "list_jobs",
    arguments: { filter: "demo" },
  });
  const mcpText = mcpCall.content[0].text;
  if (!mcpText.includes("param-demo")) {
    throw new Error(`MCP list_jobs missing param-demo: ${mcpText}`);
  }
  console.log("mcp list_jobs ok, isError:", mcpCall.isError);

  // --- 视图 / 任务 CRUD ---
  const crudView = "CRUD-View";
  await invoke("jenkins/createView", {
    connectionId: "e2e",
    name: crudView,
    description: "created by dbx e2e",
    jobs: ["demo"],
    includeRegex: "",
    recurse: false,
  });
  const createdViews = await invoke("jenkins/views", { connectionId: "e2e" });
  if (!createdViews.views.some((view) => view.name === crudView)) {
    throw new Error(`view ${crudView} was not created`);
  }
  const crudViewJobs = await invoke("jenkins/jobs", { connectionId: "e2e", view: crudView });
  console.log("CRUD view jobs:", crudViewJobs.jobs.map((job) => job.name).join(", "));
  if (crudViewJobs.jobs.length !== 1 || crudViewJobs.jobs[0].name !== "demo") {
    throw new Error(`CRUD view jobs mismatch: ${JSON.stringify(crudViewJobs.jobs)}`);
  }

  const viewConfig = await invoke("jenkins/viewConfig", { connectionId: "e2e", view: crudView });
  if (!viewConfig.fields || viewConfig.fields.name !== crudView) {
    throw new Error(`viewConfig fields missing: ${JSON.stringify(viewConfig.fields)}`);
  }
  await invoke("jenkins/updateView", {
    connectionId: "e2e",
    view: crudView,
    fields: { description: "updated by dbx", jobNames: ["demo", "param-demo"] },
  });
  const updatedView = await invoke("jenkins/viewConfig", { connectionId: "e2e", view: crudView });
  const updatedViewJobs = updatedView.fields?.jobNames ?? [];
  console.log(
    "CRUD view updated:",
    JSON.stringify(updatedView.fields?.description),
    updatedViewJobs.join(","),
  );
  if (updatedView.fields?.description !== "updated by dbx" || updatedViewJobs.length !== 2) {
    throw new Error(`view update failed: ${JSON.stringify(updatedView.fields)}`);
  }

  await invoke("jenkins/createJob", {
    connectionId: "e2e",
    name: "crud-free",
    kind: "freestyle",
    description: "created by dbx",
  });
  const freeConfig = await invoke("jenkins/jobConfig", { connectionId: "e2e", job: "crud-free" });
  if (
    freeConfig.fields?.kind !== "freestyle" ||
    freeConfig.fields?.description !== "created by dbx"
  ) {
    throw new Error(`crud-free fields mismatch: ${JSON.stringify(freeConfig.fields)}`);
  }
  await invoke("jenkins/updateJob", {
    connectionId: "e2e",
    job: "crud-free",
    fields: { description: "updated by dbx", disabled: true },
  });
  const freeUpdated = await invoke("jenkins/jobConfig", {
    connectionId: "e2e",
    job: "crud-free",
  });
  if (
    freeUpdated.fields?.disabled !== true ||
    freeUpdated.fields?.description !== "updated by dbx"
  ) {
    throw new Error(`crud-free update failed: ${JSON.stringify(freeUpdated.fields)}`);
  }
  console.log("CRUD job updated:", JSON.stringify(freeUpdated.fields));

  const renamed = await invoke("jenkins/updateJob", {
    connectionId: "e2e",
    job: "crud-free",
    fields: { title: "crud-free-renamed" },
  });
  console.log("CRUD job renamed:", JSON.stringify(renamed));
  if (renamed.fullName !== "crud-free-renamed") {
    throw new Error(`rename failed: ${JSON.stringify(renamed)}`);
  }

  let pipelineCreated = false;
  try {
    await invoke("jenkins/createJob", {
      connectionId: "e2e",
      name: "crud-pipe",
      kind: "pipeline",
      description: "created by dbx",
      script: 'echo "crud ok"',
    });
    pipelineCreated = true;
  } catch (error) {
    console.log("CRUD pipeline skipped (workflow plugin unavailable):", error.message);
  }

  if (pipelineCreated) {
    const pipeConfig = await invoke("jenkins/jobConfig", {
      connectionId: "e2e",
      job: "crud-pipe",
    });
    if (
      pipeConfig.fields?.kind !== "pipeline" ||
      !String(pipeConfig.fields?.script ?? "").includes("crud ok")
    ) {
      throw new Error(`crud-pipe fields mismatch: ${JSON.stringify(pipeConfig.fields)}`);
    }
    const beforePipe = (await latestBuild("crud-pipe"))?.number ?? 0;
    await invoke("jenkins/trigger", { connectionId: "e2e", job: "crud-pipe" });
    const crudPipeBuild = await waitFor(
      "crud-pipe build to finish",
      async () => {
        const latest = await latestBuild("crud-pipe");
        return latest && latest.number > beforePipe && latest.status === "success" ? latest : null;
      },
      90000,
    );
    console.log("CRUD pipeline build:", JSON.stringify(crudPipeBuild));

    await invoke("jenkins/saveJob", {
      connectionId: "e2e",
      job: "crud-pipe",
      xml: pipeConfig.xml,
    });
  }

  if (pipelineCreated) {
    await invoke("jenkins/deleteJob", { connectionId: "e2e", job: "crud-pipe" });
  }
  await invoke("jenkins/deleteJob", { connectionId: "e2e", job: "crud-free-renamed" });
  await invoke("jenkins/deleteView", { connectionId: "e2e", view: crudView });
  const finalJobs = await invoke("jenkins/jobs", { connectionId: "e2e" });
  const finalViews = await invoke("jenkins/views", { connectionId: "e2e" });
  if (finalJobs.jobs.some((job) => job.name.startsWith("crud-"))) {
    throw new Error("CRUD jobs were not deleted");
  }
  if (finalViews.views.some((view) => view.name === crudView)) {
    throw new Error("CRUD view was not deleted");
  }
  console.log("CRUD cleanup ok");

  await invoke("connection/connect", connectionPayload("ro", true));
  const readOnlyResponse = await call(nextId++, "jenkins/trigger", {
    connectionId: "ro",
    job: "demo",
  });
  console.log("read-only trigger:", JSON.stringify(readOnlyResponse.error));
  if (!readOnlyResponse.error?.message?.includes("read_only")) {
    throw new Error("read-only guard did not trigger");
  }

  const readOnlyCreate = await call(nextId++, "jenkins/createJob", {
    connectionId: "ro",
    name: "ro-job",
    kind: "freestyle",
  });
  console.log("read-only createJob:", JSON.stringify(readOnlyCreate.error));
  if (!readOnlyCreate.error?.message?.includes("read_only")) {
    throw new Error("read-only guard did not block createJob");
  }

  const readOnlyDeleteView = await call(nextId++, "jenkins/deleteView", {
    connectionId: "ro",
    view: "Builds",
  });
  console.log("read-only deleteView:", JSON.stringify(readOnlyDeleteView.error));
  if (!readOnlyDeleteView.error?.message?.includes("read_only")) {
    throw new Error("read-only guard did not block deleteView");
  }

  await invoke("connection/disconnect", { connection: { id: "ro" } });

  await invoke("connection/disconnect", { connection: { id: "e2e" } });
  console.log("E2E OK");
  child.stdin.end();
  child.kill();
})().catch(async (error) => {
  console.error("E2E FAILED:", error.message);
  try {
    await invoke("connection/disconnect", { connection: { id: "e2e" } });
  } catch {}
  child.kill();
  process.exit(1);
});
