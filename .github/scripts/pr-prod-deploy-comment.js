// Post merge hint on PR: manual prod deploy via Tekton comment commands.
const pr = context.payload.pull_request;
const { owner, repo } = context.repo;

const body = [
  '## Prod 部署（手动，merge 后）',
  '',
  '**合并到 `main` 不会自动部署 prod。** 需要发布时，在本 PR 下评论（每条命令单独一行）：',
  '',
  '```',
  '/deploy-cloudrun-prod-backend',
  '```',
  '',
  '```',
  '/deploy-cloudrun-prod-frontend',
  '```',
  '',
  '- 仅改 backend 时发第一条；仅改 frontend 时发第二条；都改可两条都发。',
  '- Tekton PAC 会构建镜像并 `gcloud run deploy` 到 prod（服务须已存在）。',
  '- Dev 环境：feature 分支 push 自动部署，或 PR 评论 `/deploy-cloudrun-dev`。',
  '',
  `文档：[.tekton/README.md](https://github.com/${owner}/${repo}/blob/main/.tekton/README.md)`,
].join('\n');

await github.rest.issues.createComment({
  owner,
  repo,
  issue_number: pr.number,
  body,
});
