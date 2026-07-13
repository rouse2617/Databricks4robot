# Design — CYB-3387

## 方案:纯文案统一

**候选 A(采用)**:「重置」→「清除全部」,与 chips row 一致
- 语义准确:两处按钮的 dispatch 都是 `CLEAR_ALL_FILTERS`
- "清除全部"比"重置"歧义低 —"重置"可能被理解为包括 sort / columns / view mode

**候选 B(不采用)**:反向,「清除全部」→「重置」
- 语义更宽泛,不利于用户判断

**候选 C(不采用)**:删掉侧栏 header 的两个「重置」按钮,只留 chips row 的「清除全部」
- 更激进,但侧栏 header 的按钮在 filter 侧栏打开状态下是**就近入口**,删掉会让高频操作变绕
- 保留 button 但对齐 label 更平衡

## 修改文件
- `Frontend/src/pages/AssetsPage.tsx` — 两处 button label

## Rollback
git revert 单 commit 即可。
