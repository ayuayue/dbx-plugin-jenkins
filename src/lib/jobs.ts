import type { JobSummary } from "../types";

export interface JobGroups {
  folders: JobSummary[];
  jobs: JobSummary[];
}

/** 把视图/文件夹里的任务拆成「文件夹」「任务」两组，保留后端排序。 */
export function groupJobs(list: JobSummary[]): JobGroups {
  const folders: JobSummary[] = [];
  const jobs: JobSummary[] = [];
  for (const job of list) {
    if (job.isFolder) {
      folders.push(job);
    } else {
      jobs.push(job);
    }
  }
  return { folders, jobs };
}
