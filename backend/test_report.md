# Bigtable API 测试报告

- 日期: 2026-04-25 14:14:21
- 后端: http://localhost:8080
- 耗时: 12m54s
- 总计: **984** pass / **16** fail / **1000** total
- 通过率: **98.4%**

## 维度汇总

| 维度　　　　　　　　| Pass | Fail | Total | 通过率 |
| ---------------------| ------| ------| -------| --------|
| ✅ 1.输入校验-Create | 66   | 0    | 66    | 100%　 |
| ❌ 2.Tag校验　　　　 | 77   | 1    | 78    | 99%　　|
| ✅ 3.CRUD一致性　　　| 140  | 0    | 140   | 100%　 |
| ❌ 4.算法状态机　　　| 62   | 3    | 65    | 95%　　|
| ✅ 5.依赖链　　　　　| 10   | 0    | 10    | 100%　 |
| ✅ 6.幂等性　　　　　| 40   | 0    | 40    | 100%　 |
| ✅ 7.快速循环　　　　| 5    | 0    | 5     | 100%　 |
| ✅ 8.并发　　　　　　| 11   | 0    | 11    | 100%　 |
| ✅ 9.Unicode特殊字符 | 50   | 0    | 50    | 100%　 |
| ❌ 10.分页边界　　　 | 19   | 1    | 20    | 95%　　|
| ✅ 11.Delivery　　　 | 50   | 0    | 50    | 100%　 |
| ✅ 12.CommitSegments | 11   | 0    | 11    | 100%　 |
| ✅ 13.错误响应格式　 | 20   | 0    | 20    | 100%　 |
| ✅ 14.HTTP方法路由　 | 15   | 0    | 15    | 100%　 |
| ❌ 15.认证　　　　　 | 18   | 5    | 23    | 78%　　|
| ✅ 16.AlgoEvents　　 | 5    | 0    | 5     | 100%　 |
| ❌ 17.安全　　　　　 | 14   | 6    | 20    | 70%　　|
| ✅ 18.大数据量　　　 | 5    | 0    | 5     | 100%　 |
| ✅ 19.空值处理　　　 | 15   | 0    | 15    | 100%　 |
| ✅ 20.随机Fuzz　　　 | 351  | 0    | 351   | 100%　 |

## 延迟分布

| 指标 | 值 |
|------|----|
| 请求数 | 988 |
| 平均 | 270ms |
| P50 | 218ms |
| P95 | 473ms |
| P99 | 1.909s |
| Max | 4.42s |

## 失败用例详情

| #   | 维度　　　　 | 用例　　　　　　　　　　　　　　　　　| 期望 | 实际 | 延迟　|
| -----| --------------| ---------------------------------------| ------| ------| -------|
| 1   | 2.Tag校验　　| 组合#4　　　　　　　　　　　　　　　　| 201　| 422　| 0s　　|
| 2   | 4.算法状态机 | hand_tracking@1.2.0 pending→finish　　| 409　| 422　| 0s　　|
| 3   | 4.算法状态机 | hand_tracking@1.2.0 ok→finish　　　　 | 409　| 422　| 0s　　|
| 4   | 4.算法状态机 | blocked finish　　　　　　　　　　　　| 409　| 422　| 1ms　 |
| 5   | 10.分页边界　| 分页 空格　　　　　　　　　　　　　　 | 200　| 400　| 0s　　|
| 6   | 15.认证　　　| token=" dev-token"　　　　　　　　　　| 401　| 404　| 215ms |
| 7   | 15.认证　　　| token="dev-token "　　　　　　　　　　| 401　| 404　| 214ms |
| 8   | 15.认证　　　| token="\x00\x01\x02"　　　　　　　　　| 401　| 0　　| 0s　　|
| 9   | 15.认证　　　| token="dev-token\n"　　　　　　　　　 | 401　| 0　　| 0s　　|
| 10  | 15.认证　　　| token="dev-token\r\n"　　　　　　　　 | 401　| 0　　| 0s　　|
| 11  | 17.安全　　　| 注入 "/api/v1/assets/\x00null"　　　　| 0　　| 0　　| 0s　　|
| 12  | 17.安全　　　| 超长URL len=5000　　　　　　　　　　　| 0　　| 500　| 217ms |
| 13  | 17.安全　　　| 超长URL len=10000　　　　　　　　　　 | 0　　| 500　| 222ms |
| 14  | 17.安全　　　| header注入 "\r\nX-Injected: true"　　 | 0　　| 0　　| 0s　　|
| 15  | 17.安全　　　| header注入 "\x00\x01"　　　　　　　　 | 0　　| 0　　| 0s　　|
| 16  | 17.安全　　　| header注入 "value\nAnother-Header..." | 0　　| 0　　| 0s　　|

## 全部用例明细

<details>
<summary>展开查看全部 1000 个用例</summary>

| #    | 状态 | 维度　　　　　　　| 用例　　　　　　　　　　　　　　　　　　　　　　　　　　　　| HTTP | 延迟　 |
| ------| ------| -------------------| -------------------------------------------------------------| ------| --------|
| 1    | ✅　　| 1.输入校验-Create | 必填组合 mask=0 miss=[mcap_file_id,start_timestamp_ns,en... | 400  | 2ms　　|
| 2    | ✅　　| 1.输入校验-Create | 必填组合 mask=1 miss=[start_timestamp_ns,end_timestamp_n... | 400  | 0s　　 |
| 3    | ✅　　| 1.输入校验-Create | 必填组合 mask=2 miss=[mcap_file_id,end_timestamp_ns,revi... | 400  | 0s　　 |
| 4    | ✅　　| 1.输入校验-Create | 必填组合 mask=3 miss=[end_timestamp_ns,reviewer]　　　　　　| 400  | 0s　　 |
| 5    | ✅　　| 1.输入校验-Create | 必填组合 mask=4 miss=[mcap_file_id,start_timestamp_ns,re... | 400  | 0s　　 |
| 6    | ✅　　| 1.输入校验-Create | 必填组合 mask=5 miss=[start_timestamp_ns,reviewer]　　　　　| 400  | 0s　　 |
| 7    | ✅　　| 1.输入校验-Create | 必填组合 mask=6 miss=[mcap_file_id,reviewer]　　　　　　　　| 400  | 0s　　 |
| 8    | ✅　　| 1.输入校验-Create | 必填组合 mask=7 miss=[reviewer]　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 9    | ✅　　| 1.输入校验-Create | 必填组合 mask=8 miss=[mcap_file_id,start_timestamp_ns,en... | 400  | 0s　　 |
| 10   | ✅　　| 1.输入校验-Create | 必填组合 mask=9 miss=[start_timestamp_ns,end_timestamp_n... | 400  | 0s　　 |
| 11   | ✅　　| 1.输入校验-Create | 必填组合 mask=10 miss=[mcap_file_id,end_timestamp_ns]　　　 | 400  | 0s　　 |
| 12   | ✅　　| 1.输入校验-Create | 必填组合 mask=11 miss=[end_timestamp_ns]　　　　　　　　　　| 400  | 0s　　 |
| 13   | ✅　　| 1.输入校验-Create | 必填组合 mask=12 miss=[mcap_file_id,start_timestamp_ns]　　 | 400  | 0s　　 |
| 14   | ✅　　| 1.输入校验-Create | 必填组合 mask=13 miss=[start_timestamp_ns]　　　　　　　　　| 400  | 0s　　 |
| 15   | ✅　　| 1.输入校验-Create | 必填组合 mask=14 miss=[mcap_file_id]　　　　　　　　　　　　| 400  | 0s　　 |
| 16   | ✅　　| 1.输入校验-Create | 必填组合 mask=15 miss=[]　　　　　　　　　　　　　　　　　　| 201  | 3.359s |
| 17   | ✅　　| 1.输入校验-Create | ts: start==end　　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 18   | ✅　　| 1.输入校验-Create | ts: start>end　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 19   | ✅　　| 1.输入校验-Create | ts: start=1,end=2　　　　　　　　　　　　　　　　　　　　　 | 201  | 1.241s |
| 20   | ✅　　| 1.输入校验-Create | ts: 负数start　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 459ms　|
| 21   | ✅　　| 1.输入校验-Create | ts: 极大值　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 448ms　|
| 22   | ✅　　| 1.输入校验-Create | ts: 差值=1ns　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 457ms　|
| 23   | ✅　　| 1.输入校验-Create | ts: 差值=1s　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 445ms　|
| 24   | ✅　　| 1.输入校验-Create | ts: 差值=1h　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 444ms　|
| 25   | ✅　　| 1.输入校验-Create | ts: 差值=24h　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 445ms　|
| 26   | ✅　　| 1.输入校验-Create | ts: 负极大　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 454ms　|
| 27   | ✅　　| 1.输入校验-Create | 长度 reviewer=0　　　　　　　　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 28   | ✅　　| 1.输入校验-Create | 长度 reviewer=1　　　　　　　　　　　　　　　　　　　　　　 | 201  | 438ms　|
| 29   | ✅　　| 1.输入校验-Create | 长度 reviewer=10　　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 30   | ✅　　| 1.输入校验-Create | 长度 reviewer=100　　　　　　　　　　　　　　　　　　　　　 | 201  | 437ms　|
| 31   | ✅　　| 1.输入校验-Create | 长度 reviewer=500　　　　　　　　　　　　　　　　　　　　　 | 201  | 444ms　|
| 32   | ✅　　| 1.输入校验-Create | 长度 reviewer=5000　　　　　　　　　　　　　　　　　　　　　| 201  | 539ms　|
| 33   | ✅　　| 1.输入校验-Create | 长度 owner=0　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 435ms　|
| 34   | ✅　　| 1.输入校验-Create | 长度 owner=1　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.259s |
| 35   | ✅　　| 1.输入校验-Create | 长度 owner=10　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 1.249s |
| 36   | ✅　　| 1.输入校验-Create | 长度 owner=100　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.247s |
| 37   | ✅　　| 1.输入校验-Create | 长度 owner=500　　　　　　　　　　　　　　　　　　　　　　　| 201  | 444ms　|
| 38   | ✅　　| 1.输入校验-Create | 长度 owner=5000　　　　　　　　　　　　　　　　　　　　　　 | 201  | 484ms　|
| 39   | ✅　　| 1.输入校验-Create | 长度 type=0　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 450ms　|
| 40   | ✅　　| 1.输入校验-Create | 长度 type=1　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 445ms　|
| 41   | ✅　　| 1.输入校验-Create | 长度 type=10　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 444ms　|
| 42   | ✅　　| 1.输入校验-Create | 长度 type=100　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 444ms　|
| 43   | ✅　　| 1.输入校验-Create | 长度 type=500　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 441ms　|
| 44   | ✅　　| 1.输入校验-Create | 长度 type=5000　　　　　　　　　　　　　　　　　　　　　　　| 201  | 465ms　|
| 45   | ✅　　| 1.输入校验-Create | 长度 env=0　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 46   | ✅　　| 1.输入校验-Create | 长度 env=1　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 444ms　|
| 47   | ✅　　| 1.输入校验-Create | 长度 env=10　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 579ms　|
| 48   | ✅　　| 1.输入校验-Create | 长度 env=100　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 443ms　|
| 49   | ✅　　| 1.输入校验-Create | 长度 env=500　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 442ms　|
| 50   | ✅　　| 1.输入校验-Create | 长度 env=5000　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 461ms　|
| 51   | ✅　　| 1.输入校验-Create | 长度 task=0　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 443ms　|
| 52   | ✅　　| 1.输入校验-Create | 长度 task=1　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 448ms　|
| 53   | ✅　　| 1.输入校验-Create | 长度 task=10　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 446ms　|
| 54   | ✅　　| 1.输入校验-Create | 长度 task=100　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 448ms　|
| 55   | ✅　　| 1.输入校验-Create | 长度 task=500　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 455ms　|
| 56   | ✅　　| 1.输入校验-Create | 长度 task=5000　　　　　　　　　　　　　　　　　　　　　　　| 201  | 473ms　|
| 57   | ✅　　| 1.输入校验-Create | 畸形: 空　　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 58   | ✅　　| 1.输入校验-Create | 畸形: 文本　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 59   | ✅　　| 1.输入校验-Create | 畸形: XML　　　　　　　　　　　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 60   | ✅　　| 1.输入校验-Create | 畸形: 数组　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 61   | ✅　　| 1.输入校验-Create | 畸形: null　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 62   | ✅　　| 1.输入校验-Create | 畸形: 数字　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 63   | ✅　　| 1.输入校验-Create | 畸形: 布尔　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 64   | ✅　　| 1.输入校验-Create | 畸形: 深嵌套　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 65   | ✅　　| 1.输入校验-Create | 畸形: 大JSON　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 1ms　　|
| 66   | ✅　　| 1.输入校验-Create | 畸形: 控制字符　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 67   | ✅　　| 2.Tag校验　　　　 | enum priority=critical　　　　　　　　　　　　　　　　　　　| 201  | 443ms　|
| 68   | ✅　　| 2.Tag校验　　　　 | enum priority=high　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 69   | ✅　　| 2.Tag校验　　　　 | enum priority=medium　　　　　　　　　　　　　　　　　　　　| 201  | 444ms　|
| 70   | ✅　　| 2.Tag校验　　　　 | enum priority=low　　　　　　　　　　　　　　　　　　　　　 | 201  | 444ms　|
| 71   | ✅　　| 2.Tag校验　　　　 | enum quality=excellent　　　　　　　　　　　　　　　　　　　| 201  | 439ms　|
| 72   | ✅　　| 2.Tag校验　　　　 | enum quality=good　　　　　　　　　　　　　　　　　　　　　 | 201  | 441ms　|
| 73   | ✅　　| 2.Tag校验　　　　 | enum quality=acceptable　　　　　　　　　　　　　　　　　　 | 201  | 443ms　|
| 74   | ✅　　| 2.Tag校验　　　　 | enum quality=poor　　　　　　　　　　　　　　　　　　　　　 | 201  | 448ms　|
| 75   | ✅　　| 2.Tag校验　　　　 | enum quality=unusable　　　　　　　　　　　　　　　　　　　 | 201  | 448ms　|
| 76   | ✅　　| 2.Tag校验　　　　 | enum scene=indoor　　　　　　　　　　　　　　　　　　　　　 | 201  | 443ms　|
| 77   | ✅　　| 2.Tag校验　　　　 | enum scene=outdoor　　　　　　　　　　　　　　　　　　　　　| 201  | 440ms　|
| 78   | ✅　　| 2.Tag校验　　　　 | enum scene=warehouse　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 79   | ✅　　| 2.Tag校验　　　　 | enum scene=office　　　　　　　　　　　　　　　　　　　　　 | 201  | 442ms　|
| 80   | ✅　　| 2.Tag校验　　　　 | enum scene=factory　　　　　　　　　　　　　　　　　　　　　| 201  | 438ms　|
| 81   | ✅　　| 2.Tag校验　　　　 | 非法 priority=""　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 82   | ✅　　| 2.Tag校验　　　　 | 非法 priority="INVALID"　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 83   | ✅　　| 2.Tag校验　　　　 | 非法 priority="123"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 84   | ✅　　| 2.Tag校验　　　　 | 非法 priority="true"　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 85   | ✅　　| 2.Tag校验　　　　 | 非法 priority="null"　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 86   | ✅　　| 2.Tag校验　　　　 | 非法 quality=""　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 87   | ✅　　| 2.Tag校验　　　　 | 非法 quality="INVALID"　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 88   | ✅　　| 2.Tag校验　　　　 | 非法 quality="123"　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 89   | ✅　　| 2.Tag校验　　　　 | 非法 quality="true"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 90   | ✅　　| 2.Tag校验　　　　 | 非法 quality="null"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 91   | ✅　　| 2.Tag校验　　　　 | 非法 scene=""　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 92   | ✅　　| 2.Tag校验　　　　 | 非法 scene="INVALID"　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 93   | ✅　　| 2.Tag校验　　　　 | 非法 scene="123"　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 94   | ✅　　| 2.Tag校验　　　　 | 非法 scene="true"　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 95   | ✅　　| 2.Tag校验　　　　 | 非法 scene="null"　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 96   | ✅　　| 2.Tag校验　　　　 | str task="normal"　　　　　　　　　　　　　　　　　　　　　 | 201  | 442ms　|
| 97   | ✅　　| 2.Tag校验　　　　 | str task=""　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 447ms　|
| 98   | ✅　　| 2.Tag校验　　　　 | str task="中文"　　　　　　　　　　　　　　　　　　　　　　 | 201  | 440ms　|
| 99   | ✅　　| 2.Tag校验　　　　 | str task="emoji🤖"　　　　　　　　　　　　　　　　　　　　　| 201  | 444ms　|
| 100  | ✅　　| 2.Tag校验　　　　 | str task="xxxxxxxxxxxxxxx..."　　　　　　　　　　　　　　　 | 201  | 439ms　|
| 101  | ✅　　| 2.Tag校验　　　　 | str task="长长长长长..."　　　　　　　　　　　　　　　　　　| 201  | 452ms　|
| 102  | ✅　　| 2.Tag校验　　　　 | str task="a b c"　　　　　　　　　　　　　　　　　　　　　　| 201  | 460ms　|
| 103  | ✅　　| 2.Tag校验　　　　 | str task="k=v&f=b"　　　　　　　　　　　　　　　　　　　　　| 201  | 447ms　|
| 104  | ✅　　| 2.Tag校验　　　　 | str batch="normal"　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 105  | ✅　　| 2.Tag校验　　　　 | str batch=""　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 106  | ✅　　| 2.Tag校验　　　　 | str batch="中文"　　　　　　　　　　　　　　　　　　　　　　| 201  | 445ms　|
| 107  | ✅　　| 2.Tag校验　　　　 | str batch="emoji🤖"　　　　　　　　　　　　　　　　　　　　 | 201  | 442ms　|
| 108  | ✅　　| 2.Tag校验　　　　 | str batch="xxxxxxxxxxxxxxx..."　　　　　　　　　　　　　　　| 201  | 442ms　|
| 109  | ✅　　| 2.Tag校验　　　　 | str batch="长长长长长..."　　　　　　　　　　　　　　　　　 | 201  | 444ms　|
| 110  | ✅　　| 2.Tag校验　　　　 | str batch="a b c"　　　　　　　　　　　　　　　　　　　　　 | 201  | 441ms　|
| 111  | ✅　　| 2.Tag校验　　　　 | str batch="k=v&f=b"　　　　　　　　　　　　　　　　　　　　 | 201  | 443ms　|
| 112  | ✅　　| 2.Tag校验　　　　 | str notes="normal"　　　　　　　　　　　　　　　　　　　　　| 201  | 448ms　|
| 113  | ✅　　| 2.Tag校验　　　　 | str notes=""　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 114  | ✅　　| 2.Tag校验　　　　 | str notes="中文"　　　　　　　　　　　　　　　　　　　　　　| 201  | 449ms　|
| 115  | ✅　　| 2.Tag校验　　　　 | str notes="emoji🤖"　　　　　　　　　　　　　　　　　　　　 | 201  | 442ms　|
| 116  | ✅　　| 2.Tag校验　　　　 | str notes="xxxxxxxxxxxxxxx..."　　　　　　　　　　　　　　　| 201  | 444ms　|
| 117  | ✅　　| 2.Tag校验　　　　 | str notes="长长长长长..."　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 118  | ✅　　| 2.Tag校验　　　　 | str notes="a b c"　　　　　　　　　　　　　　　　　　　　　 | 201  | 439ms　|
| 119  | ✅　　| 2.Tag校验　　　　 | str notes="k=v&f=b"　　　　　　　　　　　　　　　　　　　　 | 201  | 448ms　|
| 120  | ✅　　| 2.Tag校验　　　　 | 未注册 "fake"　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 121  | ✅　　| 2.Tag校验　　　　 | 未注册 "source"　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 122  | ✅　　| 2.Tag校验　　　　 | 未注册 "label"　　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 123  | ✅　　| 2.Tag校验　　　　 | 未注册 "category"　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 124  | ✅　　| 2.Tag校验　　　　 | 未注册 "region"　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 125  | ✅　　| 2.Tag校验　　　　 | 未注册 "team"　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 126  | ✅　　| 2.Tag校验　　　　 | 未注册 "version"　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 127  | ✅　　| 2.Tag校验　　　　 | 未注册 "status"　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 128  | ✅　　| 2.Tag校验　　　　 | 未注册 "id"　　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 129  | ✅　　| 2.Tag校验　　　　 | 未注册 "created_at"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 130  | ✅　　| 2.Tag校验　　　　 | 未注册 "updated_at"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 131  | ✅　　| 2.Tag校验　　　　 | 未注册 "mcap_file_id"　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 132  | ✅　　| 2.Tag校验　　　　 | 未注册 "asset_id"　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 133  | ✅　　| 2.Tag校验　　　　 | 未注册 "is_deleted"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 134  | ✅　　| 2.Tag校验　　　　 | 未注册 "password"　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 135  | ✅　　| 2.Tag校验　　　　 | 未注册 "secret"　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 136  | ✅　　| 2.Tag校验　　　　 | 未注册 "token"　　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 137  | ✅　　| 2.Tag校验　　　　 | 未注册 "admin"　　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 138  | ✅　　| 2.Tag校验　　　　 | 未注册 "root"　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 139  | ✅　　| 2.Tag校验　　　　 | 未注册 "DROP TABLE"　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 140  | ✅　　| 2.Tag校验　　　　 | 组合#1　　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 440ms　|
| 141  | ✅　　| 2.Tag校验　　　　 | 组合#2　　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 442ms　|
| 142  | ✅　　| 2.Tag校验　　　　 | 组合#3　　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 453ms　|
| 143  | ❌　　| 2.Tag校验　　　　 | 组合#4　　　　　　　　　　　　　　　　　　　　　　　　　　　| 422  | 0s　　 |
| 144  | ✅　　| 2.Tag校验　　　　 | 组合#5　　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 440ms　|
| 145  | ✅　　| 3.CRUD一致性　　　| 写后读#0　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 255ms　|
| 146  | ✅　　| 3.CRUD一致性　　　| 写后读#1　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 254ms　|
| 147  | ✅　　| 3.CRUD一致性　　　| 写后读#2　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 148  | ✅　　| 3.CRUD一致性　　　| 写后读#3　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 223ms　|
| 149  | ✅　　| 3.CRUD一致性　　　| 写后读#4　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 150  | ✅　　| 3.CRUD一致性　　　| 写后读#5　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 248ms　|
| 151  | ✅　　| 3.CRUD一致性　　　| 写后读#6　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 228ms　|
| 152  | ✅　　| 3.CRUD一致性　　　| 写后读#7　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 153  | ✅　　| 3.CRUD一致性　　　| 写后读#8　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 226ms　|
| 154  | ✅　　| 3.CRUD一致性　　　| 写后读#9　　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 155  | ✅　　| 3.CRUD一致性　　　| 写后读#10　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 156  | ✅　　| 3.CRUD一致性　　　| 写后读#11　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 229ms　|
| 157  | ✅　　| 3.CRUD一致性　　　| 写后读#12　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 158  | ✅　　| 3.CRUD一致性　　　| 写后读#13　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 159  | ✅　　| 3.CRUD一致性　　　| 写后读#14　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 160  | ✅　　| 3.CRUD一致性　　　| 写后读#15　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 161  | ✅　　| 3.CRUD一致性　　　| 写后读#16　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 162  | ✅　　| 3.CRUD一致性　　　| 写后读#17　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 250ms　|
| 163  | ✅　　| 3.CRUD一致性　　　| 写后读#18　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 164  | ✅　　| 3.CRUD一致性　　　| 写后读#19　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 165  | ✅　　| 3.CRUD一致性　　　| 写后读#20　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 234ms　|
| 166  | ✅　　| 3.CRUD一致性　　　| 写后读#21　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 167  | ✅　　| 3.CRUD一致性　　　| 写后读#22　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 168  | ✅　　| 3.CRUD一致性　　　| 写后读#23　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 169  | ✅　　| 3.CRUD一致性　　　| 写后读#24　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 227ms　|
| 170  | ✅　　| 3.CRUD一致性　　　| 写后读#25　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 171  | ✅　　| 3.CRUD一致性　　　| 写后读#26　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 172  | ✅　　| 3.CRUD一致性　　　| 写后读#27　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 173  | ✅　　| 3.CRUD一致性　　　| 写后读#28　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 174  | ✅　　| 3.CRUD一致性　　　| 写后读#29　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 175  | ✅　　| 3.CRUD一致性　　　| 写后读#30　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 227ms　|
| 176  | ✅　　| 3.CRUD一致性　　　| 写后读#31　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 177  | ✅　　| 3.CRUD一致性　　　| 写后读#32　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 224ms　|
| 178  | ✅　　| 3.CRUD一致性　　　| 写后读#33　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 229ms　|
| 179  | ✅　　| 3.CRUD一致性　　　| 写后读#34　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 230ms　|
| 180  | ✅　　| 3.CRUD一致性　　　| 写后读#35　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 181  | ✅　　| 3.CRUD一致性　　　| 写后读#36　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 182  | ✅　　| 3.CRUD一致性　　　| 写后读#37　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 183  | ✅　　| 3.CRUD一致性　　　| 写后读#38　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 184  | ✅　　| 3.CRUD一致性　　　| 写后读#39　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 185  | ✅　　| 3.CRUD一致性　　　| 写后读#40　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 223ms　|
| 186  | ✅　　| 3.CRUD一致性　　　| 写后读#41　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 229ms　|
| 187  | ✅　　| 3.CRUD一致性　　　| 写后读#42　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 188  | ✅　　| 3.CRUD一致性　　　| 写后读#43　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 189  | ✅　　| 3.CRUD一致性　　　| 写后读#44　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 190  | ✅　　| 3.CRUD一致性　　　| 写后读#45　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 191  | ✅　　| 3.CRUD一致性　　　| 写后读#46　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 192  | ✅　　| 3.CRUD一致性　　　| 写后读#47　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 217ms　|
| 193  | ✅　　| 3.CRUD一致性　　　| 写后读#48　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 233ms　|
| 194  | ✅　　| 3.CRUD一致性　　　| 写后读#49　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 195  | ✅　　| 3.CRUD一致性　　　| 部分更新#0　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 196  | ✅　　| 3.CRUD一致性　　　| 部分更新#1　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 197  | ✅　　| 3.CRUD一致性　　　| 部分更新#2　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 198  | ✅　　| 3.CRUD一致性　　　| 部分更新#3　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 199  | ✅　　| 3.CRUD一致性　　　| 部分更新#4　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 225ms　|
| 200  | ✅　　| 3.CRUD一致性　　　| 部分更新#5　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 201  | ✅　　| 3.CRUD一致性　　　| 部分更新#6　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 256ms　|
| 202  | ✅　　| 3.CRUD一致性　　　| 部分更新#7　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 250ms　|
| 203  | ✅　　| 3.CRUD一致性　　　| 部分更新#8　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 204  | ✅　　| 3.CRUD一致性　　　| 部分更新#9　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 205  | ✅　　| 3.CRUD一致性　　　| 部分更新#10　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 206  | ✅　　| 3.CRUD一致性　　　| 部分更新#11　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 207  | ✅　　| 3.CRUD一致性　　　| 部分更新#12　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 208  | ✅　　| 3.CRUD一致性　　　| 部分更新#13　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 209  | ✅　　| 3.CRUD一致性　　　| 部分更新#14　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 223ms　|
| 210  | ✅　　| 3.CRUD一致性　　　| 部分更新#15　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 211  | ✅　　| 3.CRUD一致性　　　| 部分更新#16　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 212  | ✅　　| 3.CRUD一致性　　　| 部分更新#17　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 223ms　|
| 213  | ✅　　| 3.CRUD一致性　　　| 部分更新#18　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 214  | ✅　　| 3.CRUD一致性　　　| 部分更新#19　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 215  | ✅　　| 3.CRUD一致性　　　| 部分更新#20　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 216  | ✅　　| 3.CRUD一致性　　　| 部分更新#21　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 217  | ✅　　| 3.CRUD一致性　　　| 部分更新#22　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 227ms　|
| 218  | ✅　　| 3.CRUD一致性　　　| 部分更新#23　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 219  | ✅　　| 3.CRUD一致性　　　| 部分更新#24　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 220  | ✅　　| 3.CRUD一致性　　　| 部分更新#25　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 221  | ✅　　| 3.CRUD一致性　　　| 部分更新#26　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 222  | ✅　　| 3.CRUD一致性　　　| 部分更新#27　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 223  | ✅　　| 3.CRUD一致性　　　| 部分更新#28　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 224  | ✅　　| 3.CRUD一致性　　　| 部分更新#29　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 225  | ✅　　| 3.CRUD一致性　　　| 版本递增#0 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 226  | ✅　　| 3.CRUD一致性　　　| 版本递增#1 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 246ms　|
| 227  | ✅　　| 3.CRUD一致性　　　| 版本递增#2 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 231ms　|
| 228  | ✅　　| 3.CRUD一致性　　　| 版本递增#3 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 229  | ✅　　| 3.CRUD一致性　　　| 版本递增#4 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 215ms　|
| 230  | ✅　　| 3.CRUD一致性　　　| 版本递增#5 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 217ms　|
| 231  | ✅　　| 3.CRUD一致性　　　| 版本递增#6 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 232  | ✅　　| 3.CRUD一致性　　　| 版本递增#7 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 233  | ✅　　| 3.CRUD一致性　　　| 版本递增#8 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 234  | ✅　　| 3.CRUD一致性　　　| 版本递增#9 expect=6 got=6　　　　　　　　　　　　　　　　　 | 200  | 217ms　|
| 235  | ✅　　| 3.CRUD一致性　　　| 版本递增#10 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 236  | ✅　　| 3.CRUD一致性　　　| 版本递增#11 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 237  | ✅　　| 3.CRUD一致性　　　| 版本递增#12 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 238  | ✅　　| 3.CRUD一致性　　　| 版本递增#13 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 239  | ✅　　| 3.CRUD一致性　　　| 版本递增#14 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 240  | ✅　　| 3.CRUD一致性　　　| 版本递增#15 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 241  | ✅　　| 3.CRUD一致性　　　| 版本递增#16 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 242  | ✅　　| 3.CRUD一致性　　　| 版本递增#17 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 243  | ✅　　| 3.CRUD一致性　　　| 版本递增#18 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 244  | ✅　　| 3.CRUD一致性　　　| 版本递增#19 expect=6 got=6　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 245  | ✅　　| 3.CRUD一致性　　　| 时间精度#0　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 225ms　|
| 246  | ✅　　| 3.CRUD一致性　　　| 时间精度#1　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 247  | ✅　　| 3.CRUD一致性　　　| 时间精度#2　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 248  | ✅　　| 3.CRUD一致性　　　| 时间精度#3　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 249  | ✅　　| 3.CRUD一致性　　　| 时间精度#4　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 223ms　|
| 250  | ✅　　| 3.CRUD一致性　　　| 时间精度#5　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 225ms　|
| 251  | ✅　　| 3.CRUD一致性　　　| 时间精度#6　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 252  | ✅　　| 3.CRUD一致性　　　| 时间精度#7　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 253  | ✅　　| 3.CRUD一致性　　　| 时间精度#8　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 254  | ✅　　| 3.CRUD一致性　　　| 时间精度#9　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 224ms　|
| 255  | ✅　　| 3.CRUD一致性　　　| 删后读#0 status=archived　　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 256  | ✅　　| 3.CRUD一致性　　　| 删后读#1 status=archived　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 257  | ✅　　| 3.CRUD一致性　　　| 删后读#2 status=archived　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 258  | ✅　　| 3.CRUD一致性　　　| 删后读#3 status=archived　　　　　　　　　　　　　　　　　　| 200  | 229ms　|
| 259  | ✅　　| 3.CRUD一致性　　　| 删后读#4 status=archived　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 260  | ✅　　| 3.CRUD一致性　　　| 删后读#5 status=archived　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 261  | ✅　　| 3.CRUD一致性　　　| 删后读#6 status=archived　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 262  | ✅　　| 3.CRUD一致性　　　| 删后读#7 status=archived　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 263  | ✅　　| 3.CRUD一致性　　　| 删后读#8 status=archived　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 264  | ✅　　| 3.CRUD一致性　　　| 删后读#9 status=archived　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 265  | ✅　　| 3.CRUD一致性　　　| 重复删除#0　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 266  | ✅　　| 3.CRUD一致性　　　| 重复删除#1　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 267  | ✅　　| 3.CRUD一致性　　　| 重复删除#2　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 268  | ✅　　| 3.CRUD一致性　　　| 重复删除#3　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 269  | ✅　　| 3.CRUD一致性　　　| 重复删除#4　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 270  | ✅　　| 3.CRUD一致性　　　| 重复删除#5　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 271  | ✅　　| 3.CRUD一致性　　　| 重复删除#6　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 272  | ✅　　| 3.CRUD一致性　　　| 重复删除#7　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 273  | ✅　　| 3.CRUD一致性　　　| 重复删除#8　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 218ms　|
| 274  | ✅　　| 3.CRUD一致性　　　| 重复删除#9　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 275  | ✅　　| 3.CRUD一致性　　　| 不存在GET#0　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 218ms　|
| 276  | ✅　　| 3.CRUD一致性　　　| 不存在GET#1　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 217ms　|
| 277  | ✅　　| 3.CRUD一致性　　　| 不存在GET#2　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 214ms　|
| 278  | ✅　　| 3.CRUD一致性　　　| 不存在GET#3　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 218ms　|
| 279  | ✅　　| 3.CRUD一致性　　　| 不存在GET#4　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 221ms　|
| 280  | ✅　　| 3.CRUD一致性　　　| 不存在GET#5　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 218ms　|
| 281  | ✅　　| 3.CRUD一致性　　　| 不存在GET#6　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 222ms　|
| 282  | ✅　　| 3.CRUD一致性　　　| 不存在GET#7　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 224ms　|
| 283  | ✅　　| 3.CRUD一致性　　　| 不存在GET#8　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 218ms　|
| 284  | ✅　　| 3.CRUD一致性　　　| 不存在GET#9　　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 217ms　|
| 285  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 start　　　　　　　　　　　　　　　　　　| 200  | 943ms　|
| 286  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 finish　　　　　　　　　　　　　　　　　 | 200  | 1.117s |
| 287  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 reset　　　　　　　　　　　　　　　　　　| 200  | 878ms　|
| 288  | ✅　　| 4.算法状态机　　　| hand_tracking@1.0.0 start　　　　　　　　　　　　　　　　　 | 200  | 895ms　|
| 289  | ✅　　| 4.算法状态机　　　| hand_tracking@1.0.0 finish　　　　　　　　　　　　　　　　　| 200  | 1.098s |
| 290  | ✅　　| 4.算法状态机　　　| hand_tracking@1.0.0 reset　　　　　　　　　　　　　　　　　 | 200  | 902ms　|
| 291  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 start　　　　　　　　　　　　　　　　　 | 200  | 884ms　|
| 292  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 finish　　　　　　　　　　　　　　　　　| 200  | 1.096s |
| 293  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 reset　　　　　　　　　　　　　　　　　 | 200  | 945ms　|
| 294  | ✅　　| 4.算法状态机　　　| head_tracking@1.0.0 start　　　　　　　　　　　　　　　　　 | 200  | 877ms　|
| 295  | ✅　　| 4.算法状态机　　　| head_tracking@1.0.0 finish　　　　　　　　　　　　　　　　　| 200  | 1.087s |
| 296  | ✅　　| 4.算法状态机　　　| head_tracking@1.0.0 reset　　　　　　　　　　　　　　　　　 | 200  | 882ms　|
| 297  | ✅　　| 4.算法状态机　　　| body_tracking@1.0.0 start　　　　　　　　　　　　　　　　　 | 200  | 870ms　|
| 298  | ✅　　| 4.算法状态机　　　| body_tracking@1.0.0 finish　　　　　　　　　　　　　　　　　| 200  | 1.109s |
| 299  | ✅　　| 4.算法状态机　　　| body_tracking@1.0.0 reset　　　　　　　　　　　　　　　　　 | 200  | 876ms　|
| 300  | ✅　　| 4.算法状态机　　　| deface@2.0.0 start　　　　　　　　　　　　　　　　　　　　　| 200  | 876ms　|
| 301  | ✅　　| 4.算法状态机　　　| deface@2.0.0 finish　　　　　　　　　　　　　　　　　　　　 | 200  | 1.119s |
| 302  | ✅　　| 4.算法状态机　　　| deface@2.0.0 reset　　　　　　　　　　　　　　　　　　　　　| 200  | 878ms　|
| 303  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 pending→finish　　　　　　　　　　　　　 | 409  | 220ms　|
| 304  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 pending→reset　　　　　　　　　　　　　　| 409  | 218ms　|
| 305  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 running→start　　　　　　　　　　　　　　| 409  | 250ms　|
| 306  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 running→reset　　　　　　　　　　　　　　| 409  | 220ms　|
| 307  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 ok→start　　　　　　　　　　　　　　　　 | 409  | 217ms　|
| 308  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 ok→finish　　　　　　　　　　　　　　　　| 409  | 224ms　|
| 309  | ❌　　| 4.算法状态机　　　| hand_tracking@1.2.0 pending→finish　　　　　　　　　　　　　| 422  | 0s　　 |
| 310  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 pending→reset　　　　　　　　　　　　　 | 409  | 221ms　|
| 311  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 running→start　　　　　　　　　　　　　 | 409  | 252ms　|
| 312  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 running→reset　　　　　　　　　　　　　 | 409  | 228ms　|
| 313  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 ok→start　　　　　　　　　　　　　　　　| 409  | 224ms　|
| 314  | ❌　　| 4.算法状态机　　　| hand_tracking@1.2.0 ok→finish　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 315  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 failed无reason　　　　　　　　　　　　　 | 422  | 0s　　 |
| 316  | ✅　　| 4.算法状态机　　　| env_analysis@1.0.0 failed有reason　　　　　　　　　　　　　 | 200  | 874ms　|
| 317  | ✅　　| 4.算法状态机　　　| hand_tracking@1.0.0 failed无reason　　　　　　　　　　　　　| 422  | 0s　　 |
| 318  | ✅　　| 4.算法状态机　　　| hand_tracking@1.0.0 failed有reason　　　　　　　　　　　　　| 200  | 876ms　|
| 319  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 failed无reason　　　　　　　　　　　　　| 422  | 0s　　 |
| 320  | ✅　　| 4.算法状态机　　　| hand_tracking@1.2.0 failed有reason　　　　　　　　　　　　　| 200  | 870ms　|
| 321  | ✅　　| 4.算法状态机　　　| head_tracking@1.0.0 failed无reason　　　　　　　　　　　　　| 422  | 0s　　 |
| 322  | ✅　　| 4.算法状态机　　　| head_tracking@1.0.0 failed有reason　　　　　　　　　　　　　| 200  | 888ms　|
| 323  | ✅　　| 4.算法状态机　　　| body_tracking@1.0.0 failed无reason　　　　　　　　　　　　　| 422  | 0s　　 |
| 324  | ✅　　| 4.算法状态机　　　| body_tracking@1.0.0 failed有reason　　　　　　　　　　　　　| 200  | 867ms　|
| 325  | ✅　　| 4.算法状态机　　　| deface@2.0.0 failed无reason　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 326  | ✅　　| 4.算法状态机　　　| deface@2.0.0 failed有reason　　　　　　　　　　　　　　　　 | 200  | 866ms　|
| 327  | ✅　　| 4.算法状态机　　　| 非法key "fake@1.0.0"　　　　　　　　　　　　　　　　　　　　| 400  | 1ms　　|
| 328  | ✅　　| 4.算法状态机　　　| 非法key "env_analysis@9.9.9"　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 329  | ✅　　| 4.算法状态机　　　| 非法key "hand_tracking@0.0.0"　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 330  | ✅　　| 4.算法状态机　　　| 非法key ""　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 331  | ✅　　| 4.算法状态机　　　| 非法key "no_ver"　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 332  | ✅　　| 4.算法状态机　　　| 非法key "@1.0.0"　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 333  | ✅　　| 4.算法状态机　　　| 非法key "env_analysis@"　　　　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 334  | ✅　　| 4.算法状态机　　　| 非法key "env_analysis"　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 335  | ✅　　| 4.算法状态机　　　| 非法key "hand_tracking@1.0.0.0"　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 336  | ✅　　| 4.算法状态机　　　| 非法key "HAND_TRACKING@1.0.0"　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 337  | ✅　　| 4.算法状态机　　　| 非法key "env-analysis@1.0.0"　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 338  | ✅　　| 4.算法状态机　　　| 非法key "env analysis@1.0.0"　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 339  | ✅　　| 4.算法状态机　　　| 非法key "../../../etc/passwd"　　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 340  | ✅　　| 4.算法状态机　　　| 非法key "<script>"　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 341  | ✅　　| 4.算法状态机　　　| 非法key "aaaaaaaaaaaaaaaaaaaaaaaaa..."　　　　　　　　　　　| 400  | 0s　　 |
| 342  | ✅　　| 4.算法状态机　　　| 非法key "DROP TABLE"　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 343  | ✅　　| 4.算法状态机　　　| 非法key "null"　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 344  | ✅　　| 4.算法状态机　　　| 非法key "undefined"　　　　　　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 345  | ✅　　| 4.算法状态机　　　| 非法key "true"　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 346  | ✅　　| 4.算法状态机　　　| 非法key "0"　　　　　　　　　　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 347  | ✅　　| 4.算法状态机　　　| blocked start　　　　　　　　　　　　　　　　　　　　　　　 | 409  | 220ms　|
| 348  | ❌　　| 4.算法状态机　　　| blocked finish　　　　　　　　　　　　　　　　　　　　　　　| 422  | 1ms　　|
| 349  | ✅　　| 4.算法状态机　　　| blocked reset　　　　　　　　　　　　　　　　　　　　　　　 | 409  | 219ms　|
| 350  | ✅　　| 5.依赖链　　　　　| unblock#0 status=pending　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 351  | ✅　　| 5.依赖链　　　　　| unblock#1 status=pending　　　　　　　　　　　　　　　　　　| 200  | 236ms　|
| 352  | ✅　　| 5.依赖链　　　　　| unblock#2 status=pending　　　　　　　　　　　　　　　　　　| 200  | 227ms　|
| 353  | ✅　　| 5.依赖链　　　　　| unblock#3 status=pending　　　　　　　　　　　　　　　　　　| 200  | 228ms　|
| 354  | ✅　　| 5.依赖链　　　　　| unblock#4 status=pending　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 355  | ✅　　| 5.依赖链　　　　　| 部分依赖#0 status=blocked　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 356  | ✅　　| 5.依赖链　　　　　| 部分依赖#1 status=blocked　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 357  | ✅　　| 5.依赖链　　　　　| 部分依赖#2 status=blocked　　　　　　　　　　　　　　　　　 | 200  | 229ms　|
| 358  | ✅　　| 5.依赖链　　　　　| 部分依赖#3 status=blocked　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 359  | ✅　　| 5.依赖链　　　　　| 部分依赖#4 status=blocked　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 360  | ✅　　| 6.幂等性　　　　　| finish幂等#0　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 224ms　|
| 361  | ✅　　| 6.幂等性　　　　　| finish不同runid#0　　　　　　　　　　　　　　　　　　　　　 | 409  | 221ms　|
| 362  | ✅　　| 6.幂等性　　　　　| finish幂等#1　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 363  | ✅　　| 6.幂等性　　　　　| finish不同runid#1　　　　　　　　　　　　　　　　　　　　　 | 409  | 220ms　|
| 364  | ✅　　| 6.幂等性　　　　　| finish幂等#2　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 223ms　|
| 365  | ✅　　| 6.幂等性　　　　　| finish不同runid#2　　　　　　　　　　　　　　　　　　　　　 | 409  | 222ms　|
| 366  | ✅　　| 6.幂等性　　　　　| finish幂等#3　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 367  | ✅　　| 6.幂等性　　　　　| finish不同runid#3　　　　　　　　　　　　　　　　　　　　　 | 409  | 220ms　|
| 368  | ✅　　| 6.幂等性　　　　　| finish幂等#4　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 369  | ✅　　| 6.幂等性　　　　　| finish不同runid#4　　　　　　　　　　　　　　　　　　　　　 | 409  | 217ms　|
| 370  | ✅　　| 6.幂等性　　　　　| finish幂等#5　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 227ms　|
| 371  | ✅　　| 6.幂等性　　　　　| finish不同runid#5　　　　　　　　　　　　　　　　　　　　　 | 409  | 229ms　|
| 372  | ✅　　| 6.幂等性　　　　　| finish幂等#6　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 373  | ✅　　| 6.幂等性　　　　　| finish不同runid#6　　　　　　　　　　　　　　　　　　　　　 | 409  | 225ms　|
| 374  | ✅　　| 6.幂等性　　　　　| finish幂等#7　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 375  | ✅　　| 6.幂等性　　　　　| finish不同runid#7　　　　　　　　　　　　　　　　　　　　　 | 409  | 220ms　|
| 376  | ✅　　| 6.幂等性　　　　　| finish幂等#8　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 377  | ✅　　| 6.幂等性　　　　　| finish不同runid#8　　　　　　　　　　　　　　　　　　　　　 | 409  | 221ms　|
| 378  | ✅　　| 6.幂等性　　　　　| finish幂等#9　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 225ms　|
| 379  | ✅　　| 6.幂等性　　　　　| finish不同runid#9　　　　　　　　　　　　　　　　　　　　　 | 409  | 220ms　|
| 380  | ✅　　| 6.幂等性　　　　　| delivery幂等#0　　　　　　　　　　　　　　　　　　　　　　　| 201  | 214ms　|
| 381  | ✅　　| 6.幂等性　　　　　| delivery冲突#0　　　　　　　　　　　　　　　　　　　　　　　| 409  | 225ms　|
| 382  | ✅　　| 6.幂等性　　　　　| delivery幂等#1　　　　　　　　　　　　　　　　　　　　　　　| 201  | 224ms　|
| 383  | ✅　　| 6.幂等性　　　　　| delivery冲突#1　　　　　　　　　　　　　　　　　　　　　　　| 409  | 234ms　|
| 384  | ✅　　| 6.幂等性　　　　　| delivery幂等#2　　　　　　　　　　　　　　　　　　　　　　　| 201  | 216ms　|
| 385  | ✅　　| 6.幂等性　　　　　| delivery冲突#2　　　　　　　　　　　　　　　　　　　　　　　| 409  | 215ms　|
| 386  | ✅　　| 6.幂等性　　　　　| delivery幂等#3　　　　　　　　　　　　　　　　　　　　　　　| 201  | 221ms　|
| 387  | ✅　　| 6.幂等性　　　　　| delivery冲突#3　　　　　　　　　　　　　　　　　　　　　　　| 409  | 218ms　|
| 388  | ✅　　| 6.幂等性　　　　　| delivery幂等#4　　　　　　　　　　　　　　　　　　　　　　　| 201  | 215ms　|
| 389  | ✅　　| 6.幂等性　　　　　| delivery冲突#4　　　　　　　　　　　　　　　　　　　　　　　| 409  | 215ms　|
| 390  | ✅　　| 6.幂等性　　　　　| delivery幂等#5　　　　　　　　　　　　　　　　　　　　　　　| 201  | 218ms　|
| 391  | ✅　　| 6.幂等性　　　　　| delivery冲突#5　　　　　　　　　　　　　　　　　　　　　　　| 409  | 217ms　|
| 392  | ✅　　| 6.幂等性　　　　　| delivery幂等#6　　　　　　　　　　　　　　　　　　　　　　　| 201  | 217ms　|
| 393  | ✅　　| 6.幂等性　　　　　| delivery冲突#6　　　　　　　　　　　　　　　　　　　　　　　| 409  | 217ms　|
| 394  | ✅　　| 6.幂等性　　　　　| delivery幂等#7　　　　　　　　　　　　　　　　　　　　　　　| 201  | 216ms　|
| 395  | ✅　　| 6.幂等性　　　　　| delivery冲突#7　　　　　　　　　　　　　　　　　　　　　　　| 409  | 218ms　|
| 396  | ✅　　| 6.幂等性　　　　　| delivery幂等#8　　　　　　　　　　　　　　　　　　　　　　　| 201  | 218ms　|
| 397  | ✅　　| 6.幂等性　　　　　| delivery冲突#8　　　　　　　　　　　　　　　　　　　　　　　| 409  | 215ms　|
| 398  | ✅　　| 6.幂等性　　　　　| delivery幂等#9　　　　　　　　　　　　　　　　　　　　　　　| 201  | 215ms　|
| 399  | ✅　　| 6.幂等性　　　　　| delivery冲突#9　　　　　　　　　　　　　　　　　　　　　　　| 409  | 219ms　|
| 400  | ✅　　| 7.快速循环　　　　| 资产#0 循环3次 ok=3　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 401  | ✅　　| 7.快速循环　　　　| 资产#1 循环3次 ok=3　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 402  | ✅　　| 7.快速循环　　　　| 资产#2 循环3次 ok=3　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 403  | ✅　　| 7.快速循环　　　　| 资产#3 循环3次 ok=3　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 404  | ✅　　| 7.快速循环　　　　| 资产#4 循环3次 ok=3　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 405  | ✅　　| 8.并发　　　　　　| 并发start#0 200×5　　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 406  | ✅　　| 8.并发　　　　　　| 并发start#1 200×5　　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 407  | ✅　　| 8.并发　　　　　　| 并发start#2 200×5　　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 408  | ✅　　| 8.并发　　　　　　| 并发start#3 200×5　　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 409  | ✅　　| 8.并发　　　　　　| 并发start#4 200×5　　　　　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 410  | ✅　　| 8.并发　　　　　　| 并发PATCH#0　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 217ms　|
| 411  | ✅　　| 8.并发　　　　　　| 并发PATCH#1　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 226ms　|
| 412  | ✅　　| 8.并发　　　　　　| 并发PATCH#2　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 249ms　|
| 413  | ✅　　| 8.并发　　　　　　| 并发PATCH#3　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 414  | ✅　　| 8.并发　　　　　　| 并发PATCH#4　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 218ms　|
| 415  | ✅　　| 8.并发　　　　　　| 并发创建50个 ok=50　　　　　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 416  | ✅　　| 9.Unicode特殊字符 | 创建 "张三"　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 423ms　|
| 417  | ✅　　| 9.Unicode特殊字符 | 往返 "张三"　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 216ms　|
| 418  | ✅　　| 9.Unicode特殊字符 | 创建 "田中太郎"　　　　　　　　　　　　　　　　　　　　　　 | 201  | 439ms　|
| 419  | ✅　　| 9.Unicode特殊字符 | 往返 "田中太郎"　　　　　　　　　　　　　　　　　　　　　　 | 200  | 220ms　|
| 420  | ✅　　| 9.Unicode特殊字符 | 创建 "김철수"　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 427ms　|
| 421  | ✅　　| 9.Unicode特殊字符 | 往返 "김철수"　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 216ms　|
| 422  | ✅　　| 9.Unicode特殊字符 | 创建 "Müller"　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 427ms　|
| 423  | ✅　　| 9.Unicode特殊字符 | 往返 "Müller"　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 214ms　|
| 424  | ✅　　| 9.Unicode特殊字符 | 创建 "Ñoño"　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 430ms　|
| 425  | ✅　　| 9.Unicode特殊字符 | 往返 "Ñoño"　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 212ms　|
| 426  | ✅　　| 9.Unicode特殊字符 | 创建 "Ωmega"　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 424ms　|
| 427  | ✅　　| 9.Unicode特殊字符 | 往返 "Ωmega"　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 215ms　|
| 428  | ✅　　| 9.Unicode特殊字符 | 创建 "emoji🤖🚀\xf0\x9f..."　　　　　　　　　　　　　　　　　| 201  | 423ms　|
| 429  | ✅　　| 9.Unicode特殊字符 | 往返 "emoji🤖🚀\xf0\x9f..."　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 430  | ✅　　| 9.Unicode特殊字符 | 创建 "混合abc中文..."　　　　　　　　　　　　　　　　　　　 | 201  | 427ms　|
| 431  | ✅　　| 9.Unicode特殊字符 | 往返 "混合abc中文..."　　　　　　　　　　　　　　　　　　　 | 200  | 216ms　|
| 432  | ✅　　| 9.Unicode特殊字符 | 创建 "tab\there"　　　　　　　　　　　　　　　　　　　　　　| 201  | 423ms　|
| 433  | ✅　　| 9.Unicode特殊字符 | 往返 "tab\there"　　　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 434  | ✅　　| 9.Unicode特殊字符 | 创建 "newline\nhere"　　　　　　　　　　　　　　　　　　　　| 201  | 423ms　|
| 435  | ✅　　| 9.Unicode特殊字符 | 往返 "newline\nhere"　　　　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 436  | ✅　　| 9.Unicode特殊字符 | 创建 "引号\"here\""　　　　　　　　　　　　　　　　　　　　 | 201  | 423ms　|
| 437  | ✅　　| 9.Unicode特殊字符 | 往返 "引号\"here\""　　　　　　　　　　　　　　　　　　　　 | 200  | 212ms　|
| 438  | ✅　　| 9.Unicode特殊字符 | 创建 "反斜杠\\here"　　　　　　　　　　　　　　　　　　　　 | 201  | 422ms　|
| 439  | ✅　　| 9.Unicode特殊字符 | 往返 "反斜杠\\here"　　　　　　　　　　　　　　　　　　　　 | 200  | 213ms　|
| 440  | ✅　　| 9.Unicode特殊字符 | 创建 "斜杠/here"　　　　　　　　　　　　　　　　　　　　　　| 201  | 433ms　|
| 441  | ✅　　| 9.Unicode特殊字符 | 往返 "斜杠/here"　　　　　　　　　　　　　　　　　　　　　　| 200  | 213ms　|
| 442  | ✅　　| 9.Unicode特殊字符 | 创建 "尖括号<>here"　　　　　　　　　　　　　　　　　　　　 | 201  | 439ms　|
| 443  | ✅　　| 9.Unicode特殊字符 | 往返 "尖括号<>here"　　　　　　　　　　　　　　　　　　　　 | 200  | 216ms　|
| 444  | ✅　　| 9.Unicode特殊字符 | 创建 "&amp;entity"　　　　　　　　　　　　　　　　　　　　　| 201  | 423ms　|
| 445  | ✅　　| 9.Unicode特殊字符 | 往返 "&amp;entity"　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 446  | ✅　　| 9.Unicode特殊字符 | 创建 "零宽\u200b字符"　　　　　　　　　　　　　　　　　　　 | 201  | 425ms　|
| 447  | ✅　　| 9.Unicode特殊字符 | 往返 "零宽\u200b字符"　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 448  | ✅　　| 9.Unicode特殊字符 | 创建 "RTL\u200ftext"　　　　　　　　　　　　　　　　　　　　| 201  | 430ms　|
| 449  | ✅　　| 9.Unicode特殊字符 | 往返 "RTL\u200ftext"　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 450  | ✅　　| 9.Unicode特殊字符 | 创建 "BOM\ufeffhere"　　　　　　　　　　　　　　　　　　　　| 201  | 430ms　|
| 451  | ✅　　| 9.Unicode特殊字符 | 往返 "BOM\ufeffhere"　　　　　　　　　　　　　　　　　　　　| 200  | 221ms　|
| 452  | ✅　　| 9.Unicode特殊字符 | 创建 "中中中中中..."　　　　　　　　　　　　　　　　　　　　| 201  | 427ms　|
| 453  | ✅　　| 9.Unicode特殊字符 | 往返 "中中中中中..."　　　　　　　　　　　　　　　　　　　　| 200  | 230ms　|
| 454  | ✅　　| 9.Unicode特殊字符 | 创建 "🤖🤖🤖\xf0\x9f\xa4..."　　　　　　　　　　　　　　　　| 201  | 428ms　|
| 455  | ✅　　| 9.Unicode特殊字符 | 往返 "🤖🤖🤖\xf0\x9f\xa4..."　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 456  | ✅　　| 9.Unicode特殊字符 | mcap_id "file with spaces"　　　　　　　　　　　　　　　　　| 201  | 426ms　|
| 457  | ✅　　| 9.Unicode特殊字符 | mcap_id "file#hash"　　　　　　　　　　　　　　　　　　　　 | 201  | 429ms　|
| 458  | ✅　　| 9.Unicode特殊字符 | mcap_id "file@at"　　　　　　　　　　　　　　　　　　　　　 | 201  | 426ms　|
| 459  | ✅　　| 9.Unicode特殊字符 | mcap_id "file/slash"　　　　　　　　　　　　　　　　　　　　| 201  | 425ms　|
| 460  | ✅　　| 9.Unicode特殊字符 | mcap_id "file?query=1"　　　　　　　　　　　　　　　　　　　| 201  | 430ms　|
| 461  | ✅　　| 9.Unicode特殊字符 | mcap_id "file%20encoded"　　　　　　　　　　　　　　　　　　| 201  | 424ms　|
| 462  | ✅　　| 9.Unicode特殊字符 | mcap_id "file+plus"　　　　　　　　　　　　　　　　　　　　 | 201  | 425ms　|
| 463  | ✅　　| 9.Unicode特殊字符 | mcap_id "file=equals"　　　　　　　　　　　　　　　　　　　 | 201  | 429ms　|
| 464  | ✅　　| 9.Unicode特殊字符 | mcap_id "file;semicolon"　　　　　　　　　　　　　　　　　　| 201  | 426ms　|
| 465  | ✅　　| 9.Unicode特殊字符 | mcap_id "file&ampersand"　　　　　　　　　　　　　　　　　　| 201  | 429ms　|
| 466  | ✅　　| 10.分页边界　　　 | 分页 正常　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 224ms　|
| 467  | ✅　　| 10.分页边界　　　 | 分页 page=0　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 210ms　|
| 468  | ✅　　| 10.分页边界　　　 | 分页 page=-1　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 219ms　|
| 469  | ✅　　| 10.分页边界　　　 | 分页 page=999999　　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 470  | ✅　　| 10.分页边界　　　 | 分页 ps=0　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 225ms　|
| 471  | ✅　　| 10.分页边界　　　 | 分页 ps=-1　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 472  | ✅　　| 10.分页边界　　　 | 分页 ps=999　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 473  | ✅　　| 10.分页边界　　　 | 分页 ps=abc　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 215ms　|
| 474  | ✅　　| 10.分页边界　　　 | 分页 page=abc　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 208ms　|
| 475  | ✅　　| 10.分页边界　　　 | 分页 both=abc　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 209ms　|
| 476  | ✅　　| 10.分页边界　　　 | 分页 empty　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 477  | ✅　　| 10.分页边界　　　 | 分页 page=1.5　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 215ms　|
| 478  | ✅　　| 10.分页边界　　　 | 分页 ps=1e9　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 212ms　|
| 479  | ✅　　| 10.分页边界　　　 | 分页 page=0x10　　　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 480  | ✅　　| 10.分页边界　　　 | 分页 负大数　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 208ms　|
| 481  | ✅　　| 10.分页边界　　　 | 分页 ps=MAX　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 211ms　|
| 482  | ✅　　| 10.分页边界　　　 | 分页 page=MAX　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 211ms　|
| 483  | ✅　　| 10.分页边界　　　 | 分页 unicode　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 484  | ❌　　| 10.分页边界　　　 | 分页 空格　　　　　　　　　　　　　　　　　　　　　　　　　 | 400  | 0s　　 |
| 485  | ✅　　| 10.分页边界　　　 | 分页 特殊　　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 213ms　|
| 486  | ✅　　| 11.Delivery　　　 | 多资产#0　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.957s |
| 487  | ✅　　| 11.Delivery　　　 | 资产6ded6c3e..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 488  | ✅　　| 11.Delivery　　　 | 资产88bc5ffa..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 489  | ✅　　| 11.Delivery　　　 | 资产b3a35762..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 209ms　|
| 490  | ✅　　| 11.Delivery　　　 | 多资产#1　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.952s |
| 491  | ✅　　| 11.Delivery　　　 | 资产034b56e8..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 492  | ✅　　| 11.Delivery　　　 | 资产c4d3eeb8..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 493  | ✅　　| 11.Delivery　　　 | 资产cc6edf21..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 494  | ✅　　| 11.Delivery　　　 | 多资产#2　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.913s |
| 495  | ✅　　| 11.Delivery　　　 | 资产018edcac..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 496  | ✅　　| 11.Delivery　　　 | 资产387eb144..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 497  | ✅　　| 11.Delivery　　　 | 资产aba377e5..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 498  | ✅　　| 11.Delivery　　　 | 多资产#3　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.897s |
| 499  | ✅　　| 11.Delivery　　　 | 资产57c0fbf3..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 500  | ✅　　| 11.Delivery　　　 | 资产0990b63a..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 209ms　|
| 501  | ✅　　| 11.Delivery　　　 | 资产cf70e230..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 502  | ✅　　| 11.Delivery　　　 | 多资产#4　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.909s |
| 503  | ✅　　| 11.Delivery　　　 | 资产e2805b01..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 504  | ✅　　| 11.Delivery　　　 | 资产438521b0..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 213ms　|
| 505  | ✅　　| 11.Delivery　　　 | 资产665178ac..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 506  | ✅　　| 11.Delivery　　　 | 多资产#5　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.912s |
| 507  | ✅　　| 11.Delivery　　　 | 资产e295dc05..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 213ms　|
| 508  | ✅　　| 11.Delivery　　　 | 资产f163e87d..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 509  | ✅　　| 11.Delivery　　　 | 资产b7927c1e..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 510  | ✅　　| 11.Delivery　　　 | 多资产#6　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.901s |
| 511  | ✅　　| 11.Delivery　　　 | 资产4b0de25e..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 512  | ✅　　| 11.Delivery　　　 | 资产a9c8888e..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 513  | ✅　　| 11.Delivery　　　 | 资产a47d72fa..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 220ms　|
| 514  | ✅　　| 11.Delivery　　　 | 多资产#7　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.909s |
| 515  | ✅　　| 11.Delivery　　　 | 资产c76192cf..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 210ms　|
| 516  | ✅　　| 11.Delivery　　　 | 资产d72e3f96..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 212ms　|
| 517  | ✅　　| 11.Delivery　　　 | 资产c3486298..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 213ms　|
| 518  | ✅　　| 11.Delivery　　　 | 多资产#8　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.905s |
| 519  | ✅　　| 11.Delivery　　　 | 资产edda6e80..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 211ms　|
| 520  | ✅　　| 11.Delivery　　　 | 资产a06810cb..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 213ms　|
| 521  | ✅　　| 11.Delivery　　　 | 资产b6d0fcee..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 245ms　|
| 522  | ✅　　| 11.Delivery　　　 | 多资产#9　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 2.02s　|
| 523  | ✅　　| 11.Delivery　　　 | 资产f3b79e5a..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 250ms　|
| 524  | ✅　　| 11.Delivery　　　 | 资产813947bb..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 216ms　|
| 525  | ✅　　| 11.Delivery　　　 | 资产a096a3ab..交付　　　　　　　　　　　　　　　　　　　　　| 200  | 217ms　|
| 526  | ✅　　| 11.Delivery　　　 | 无idem-key#0　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 527  | ✅　　| 11.Delivery　　　 | 无idem-key#1　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 528  | ✅　　| 11.Delivery　　　 | 无idem-key#2　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 529  | ✅　　| 11.Delivery　　　 | 无idem-key#3　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 530  | ✅　　| 11.Delivery　　　 | 无idem-key#4　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 531  | ✅　　| 11.Delivery　　　 | 空assets#0　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 532  | ✅　　| 11.Delivery　　　 | 空assets#1　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 533  | ✅　　| 11.Delivery　　　 | 空assets#2　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 534  | ✅　　| 11.Delivery　　　 | 空assets#3　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 535  | ✅　　| 11.Delivery　　　 | 空assets#4　　　　　　　　　　　　　　　　　　　　　　　　　| 400  | 0s　　 |
| 536  | ✅　　| 12.CommitSegments | 1 ranges　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 440ms　|
| 537  | ✅　　| 12.CommitSegments | 2 ranges　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 883ms　|
| 538  | ✅　　| 12.CommitSegments | 3 ranges　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 1.324s |
| 539  | ✅　　| 12.CommitSegments | 5 ranges　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 2.231s |
| 540  | ✅　　| 12.CommitSegments | 10 ranges　　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 4.42s　|
| 541  | ✅　　| 12.CommitSegments | 非法range#0　　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 542  | ✅　　| 12.CommitSegments | 非法range#1　　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 543  | ✅　　| 12.CommitSegments | 非法range#2　　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 438ms　|
| 544  | ✅　　| 12.CommitSegments | 非法range#3　　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 545  | ✅　　| 12.CommitSegments | 非法range#4　　　　　　　　　　　　　　　　　　　　　　　　 | 422  | 0s　　 |
| 546  | ✅　　| 12.CommitSegments | 空ranges　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 0s　　 |
| 547  | ✅　　| 13.错误响应格式　 | 404 GET code=true msg=true rid=true　　　　　　　　　　　　 | 404  | 223ms　|
| 548  | ✅　　| 13.错误响应格式　 | 400 空body code=true msg=true rid=true　　　　　　　　　　　| 400  | 0s　　 |
| 549  | ✅　　| 13.错误响应格式　 | 401 无token code=true msg=true rid=true　　　　　　　　　　 | 401  | 0s　　 |
| 550  | ✅　　| 13.错误响应格式　 | 422 非法tag code=true msg=true rid=true　　　　　　　　　　 | 422  | 0s　　 |
| 551  | ✅　　| 13.错误响应格式　 | 400 非法algo code=true msg=true rid=true　　　　　　　　　　| 400  | 0s　　 |
| 552  | ✅　　| 13.错误响应格式　 | request_id唯一#0　　　　　　　　　　　　　　　　　　　　　　| 404  | 215ms　|
| 553  | ✅　　| 13.错误响应格式　 | request_id唯一#1　　　　　　　　　　　　　　　　　　　　　　| 404  | 234ms　|
| 554  | ✅　　| 13.错误响应格式　 | request_id唯一#2　　　　　　　　　　　　　　　　　　　　　　| 404  | 220ms　|
| 555  | ✅　　| 13.错误响应格式　 | request_id唯一#3　　　　　　　　　　　　　　　　　　　　　　| 404  | 235ms　|
| 556  | ✅　　| 13.错误响应格式　 | request_id唯一#4　　　　　　　　　　　　　　　　　　　　　　| 404  | 218ms　|
| 557  | ✅　　| 13.错误响应格式　 | request_id唯一#5　　　　　　　　　　　　　　　　　　　　　　| 404  | 216ms　|
| 558  | ✅　　| 13.错误响应格式　 | request_id唯一#6　　　　　　　　　　　　　　　　　　　　　　| 404  | 219ms　|
| 559  | ✅　　| 13.错误响应格式　 | request_id唯一#7　　　　　　　　　　　　　　　　　　　　　　| 404  | 298ms　|
| 560  | ✅　　| 13.错误响应格式　 | request_id唯一#8　　　　　　　　　　　　　　　　　　　　　　| 404  | 219ms　|
| 561  | ✅　　| 13.错误响应格式　 | request_id唯一#9　　　　　　　　　　　　　　　　　　　　　　| 404  | 215ms　|
| 562  | ✅　　| 13.错误响应格式　 | request_id唯一#10　　　　　　　　　　　　　　　　　　　　　 | 404  | 218ms　|
| 563  | ✅　　| 13.错误响应格式　 | request_id唯一#11　　　　　　　　　　　　　　　　　　　　　 | 404  | 215ms　|
| 564  | ✅　　| 13.错误响应格式　 | request_id唯一#12　　　　　　　　　　　　　　　　　　　　　 | 404  | 215ms　|
| 565  | ✅　　| 13.错误响应格式　 | request_id唯一#13　　　　　　　　　　　　　　　　　　　　　 | 404  | 217ms　|
| 566  | ✅　　| 13.错误响应格式　 | request_id唯一#14　　　　　　　　　　　　　　　　　　　　　 | 404  | 215ms　|
| 567  | ✅　　| 14.HTTP方法路由　 | PUT /assets/:id　　　　　　　　　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 568  | ✅　　| 14.HTTP方法路由　 | OPTIONS /assets/:id　　　　　　　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 569  | ✅　　| 14.HTTP方法路由　 | HEAD /assets/:id　　　　　　　　　　　　　　　　　　　　　　| 404  | 0s　　 |
| 570  | ✅　　| 14.HTTP方法路由　 | TRACE /assets/:id　　　　　　　　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 571  | ✅　　| 14.HTTP方法路由　 | CONNECT /assets/:id　　　　　　　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 572  | ✅　　| 14.HTTP方法路由　 | GET /api/v1/nonexistent　　　　　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 573  | ✅　　| 14.HTTP方法路由　 | GET /api/v2/assets　　　　　　　　　　　　　　　　　　　　　| 404  | 0s　　 |
| 574  | ✅　　| 14.HTTP方法路由　 | GET /api/v1/assets/x/y/z　　　　　　　　　　　　　　　　　　| 404  | 0s　　 |
| 575  | ✅　　| 14.HTTP方法路由　 | GET /admin　　　　　　　　　　　　　　　　　　　　　　　　　| 404  | 0s　　 |
| 576  | ✅　　| 14.HTTP方法路由　 | GET /api/v1/assets/../../../etc　　　　　　　　　　　　　　 | 404  | 0s　　 |
| 577  | ✅　　| 14.HTTP方法路由　 | healthz无token#0　　　　　　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 578  | ✅　　| 14.HTTP方法路由　 | healthz无token#1　　　　　　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 579  | ✅　　| 14.HTTP方法路由　 | healthz无token#2　　　　　　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 580  | ✅　　| 14.HTTP方法路由　 | healthz无token#3　　　　　　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 581  | ✅　　| 14.HTTP方法路由　 | healthz无token#4　　　　　　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 582  | ✅　　| 15.认证　　　　　 | token=""　　　　　　　　　　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 583  | ✅　　| 15.认证　　　　　 | token="wrong"　　　　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 584  | ✅　　| 15.认证　　　　　 | token="dev-token-x"　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 585  | ✅　　| 15.认证　　　　　 | token="DEV-TOKEN"　　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 586  | ❌　　| 15.认证　　　　　 | token=" dev-token"　　　　　　　　　　　　　　　　　　　　　| 404  | 215ms　|
| 587  | ❌　　| 15.认证　　　　　 | token="dev-token "　　　　　　　　　　　　　　　　　　　　　| 404  | 214ms　|
| 588  | ✅　　| 15.认证　　　　　 | token="xxxxxxxxxxxxxxxxxxxx..."　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 589  | ✅　　| 15.认证　　　　　 | token="null"　　　　　　　　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 590  | ✅　　| 15.认证　　　　　 | token="undefined"　　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 591  | ✅　　| 15.认证　　　　　 | token="true"　　　　　　　　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 592  | ✅　　| 15.认证　　　　　 | token="false"　　　　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 593  | ✅　　| 15.认证　　　　　 | token="0"　　　　　　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 594  | ✅　　| 15.认证　　　　　 | token="Bearer wrong"　　　　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 595  | ✅　　| 15.认证　　　　　 | token="Bearer "　　　　　　　　　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 596  | ✅　　| 15.认证　　　　　 | token="Basic dXNlcjpwYXNz"　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 597  | ✅　　| 15.认证　　　　　 | token="<script>alert(1)</sc..."　　　　　　　　　　　　　　 | 401  | 0s　　 |
| 598  | ✅　　| 15.认证　　　　　 | token="'; DROP TABLE --"　　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 599  | ❌　　| 15.认证　　　　　 | token="\x00\x01\x02"　　　　　　　　　　　　　　　　　　　　| 0    | 0s　　 |
| 600  | ❌　　| 15.认证　　　　　 | token="dev-token\n"　　　　　　　　　　　　　　　　　　　　 | 0    | 0s　　 |
| 601  | ❌　　| 15.认证　　　　　 | token="dev-token\r\n"　　　　　　　　　　　　　　　　　　　 | 0    | 0s　　 |
| 602  | ✅　　| 15.认证　　　　　 | Bearer格式　　　　　　　　　　　　　　　　　　　　　　　　　| 404  | 213ms　|
| 603  | ✅　　| 15.认证　　　　　 | Bearer错误　　　　　　　　　　　　　　　　　　　　　　　　　| 401  | 0s　　 |
| 604  | ✅　　| 15.认证　　　　　 | Auth直接token　　　　　　　　　　　　　　　　　　　　　　　 | 404  | 214ms　|
| 605  | ✅　　| 16.AlgoEvents　　 | 全部事件 count=12　　　　　　　　　　　　　　　　　　　　　 | 200  | 458ms　|
| 606  | ✅　　| 16.AlgoEvents　　 | 过滤 env_analysis@1.0.0 count=12　　　　　　　　　　　　　　| 200  | 463ms　|
| 607  | ✅　　| 16.AlgoEvents　　 | 过滤不存在key　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 500ms　|
| 608  | ✅　　| 16.AlgoEvents　　 | 不存在资产　　　　　　　　　　　　　　　　　　　　　　　　　| 404  | 211ms　|
| 609  | ✅　　| 16.AlgoEvents　　 | 过滤 hand_tracking@1.2.0　　　　　　　　　　　　　　　　　　| 200  | 458ms　|
| 610  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/../../etc/passwd"　　　　　　　　　　　| 404  | 0s　　 |
| 611  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/%2e%2e%2f%2e%2e%2fetc%2fp..."　　　　　| 404  | 0s　　 |
| 612  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/..\\..\\windows\\system32"　　　　　　 | 404  | 215ms　|
| 613  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/%00"　　　　　　　　　　　　　　　　　 | 404  | 209ms　|
| 614  | ❌　　| 17.安全　　　　　 | 注入 "/api/v1/assets/\x00null"　　　　　　　　　　　　　　　| 0    | 0s　　 |
| 615  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/<script>alert(1)</script>"　　　　　　 | 404  | 0s　　 |
| 616  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/' OR 1=1 --"　　　　　　　　　　　　　 | 404  | 216ms　|
| 617  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/; ls -la"　　　　　　　　　　　　　　　| 404  | 210ms　|
| 618  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/${jndi:ldap://evil.com}"　　　　　　　 | 404  | 0s　　 |
| 619  | ✅　　| 17.安全　　　　　 | 注入 "/api/v1/assets/{{7*7}}"　　　　　　　　　　　　　　　 | 404  | 207ms　|
| 620  | ✅　　| 17.安全　　　　　 | 超长URL len=100　　　　　　　　　　　　　　　　　　　　　　 | 404  | 211ms　|
| 621  | ✅　　| 17.安全　　　　　 | 超长URL len=500　　　　　　　　　　　　　　　　　　　　　　 | 404  | 213ms　|
| 622  | ✅　　| 17.安全　　　　　 | 超长URL len=1000　　　　　　　　　　　　　　　　　　　　　　| 404  | 214ms　|
| 623  | ❌　　| 17.安全　　　　　 | 超长URL len=5000　　　　　　　　　　　　　　　　　　　　　　| 500  | 217ms　|
| 624  | ❌　　| 17.安全　　　　　 | 超长URL len=10000　　　　　　　　　　　　　　　　　　　　　 | 500  | 222ms　|
| 625  | ❌　　| 17.安全　　　　　 | header注入 "\r\nX-Injected: true"　　　　　　　　　　　　　 | 0    | 0s　　 |
| 626  | ✅　　| 17.安全　　　　　 | header注入 "xxxxxxxxxxxxxxxxxxxx..."　　　　　　　　　　　　| 404  | 211ms　|
| 627  | ❌　　| 17.安全　　　　　 | header注入 "\x00\x01"　　　　　　　　　　　　　　　　　　　 | 0    | 0s　　 |
| 628  | ❌　　| 17.安全　　　　　 | header注入 "value\nAnother-Header..."　　　　　　　　　　　 | 0    | 0s　　 |
| 629  | ✅　　| 17.安全　　　　　 | header注入 "<script>"　　　　　　　　　　　　　　　　　　　 | 404  | 209ms　|
| 630  | ✅　　| 18.大数据量　　　 | 多algo资产#0 algos=44　　　　　　　　　　　　　　　　　　　 | 200  | 221ms　|
| 631  | ✅　　| 18.大数据量　　　 | 多algo资产#1 algos=44　　　　　　　　　　　　　　　　　　　 | 200  | 222ms　|
| 632  | ✅　　| 18.大数据量　　　 | 多algo资产#2 algos=44　　　　　　　　　　　　　　　　　　　 | 200  | 223ms　|
| 633  | ✅　　| 18.大数据量　　　 | 多algo资产#3 algos=44　　　　　　　　　　　　　　　　　　　 | 200  | 246ms　|
| 634  | ✅　　| 18.大数据量　　　 | 多algo资产#4 algos=44　　　　　　　　　　　　　　　　　　　 | 200  | 219ms　|
| 635  | ✅　　| 19.空值处理　　　 | owner=空字符串　　　　　　　　　　　　　　　　　　　　　　　| 201  | 441ms　|
| 636  | ✅　　| 19.空值处理　　　 | owner读回=""　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 214ms　|
| 637  | ✅　　| 19.空值处理　　　 | type=空字符串　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 424ms　|
| 638  | ✅　　| 19.空值处理　　　 | type读回=""　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 213ms　|
| 639  | ✅　　| 19.空值处理　　　 | env=空字符串　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 439ms　|
| 640  | ✅　　| 19.空值处理　　　 | env读回=""　　　　　　　　　　　　　　　　　　　　　　　　　| 200  | 222ms　|
| 641  | ✅　　| 19.空值处理　　　 | task=空字符串　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 420ms　|
| 642  | ✅　　| 19.空值处理　　　 | task读回=""　　　　　　　　　　　　　　　　　　　　　　　　 | 200  | 212ms　|
| 643  | ✅　　| 19.空值处理　　　 | owner不传　　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 428ms　|
| 644  | ✅　　| 19.空值处理　　　 | type不传　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 454ms　|
| 645  | ✅　　| 19.空值处理　　　 | env不传　　　　　　　　　　　　　　　　　　　　　　　　　　 | 201  | 461ms　|
| 646  | ✅　　| 19.空值处理　　　 | task不传　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 448ms　|
| 647  | ✅　　| 19.空值处理　　　 | tags不传　　　　　　　　　　　　　　　　　　　　　　　　　　| 201  | 444ms　|
| 648  | ✅　　| 19.空值处理　　　 | PATCH reviewer=空　　　　　　　　　　　　　　　　　　　　　 | 200  | 460ms　|
| 649  | ✅　　| 19.空值处理　　　 | PATCH owner=空　　　　　　　　　　　　　　　　　　　　　　　| 200  | 444ms　|
| 650  | ✅　　| 20.随机Fuzz　　　 | fuzz#0 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　　 | 200  | 215ms　|
| 651  | ✅　　| 20.随机Fuzz　　　 | fuzz#1 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 453ms　|
| 652  | ✅　　| 20.随机Fuzz　　　 | fuzz#2 POST /api/v1/assets → 201　　　　　　　　　　　　　　| 201  | 441ms　|
| 653  | ✅　　| 20.随机Fuzz　　　 | fuzz#3 DELETE /api/v1/assets/fuzz-64616 → 200　　　　　　　 | 200  | 215ms　|
| 654  | ✅　　| 20.随机Fuzz　　　 | fuzz#4 POST /api/v1/assets → 201　　　　　　　　　　　　　　| 201  | 435ms　|
| 655  | ✅　　| 20.随机Fuzz　　　 | fuzz#5 GET /healthz → 200　　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 656  | ✅　　| 20.随机Fuzz　　　 | fuzz#6 GET /api/v1/assets/fuzz-577626 → 404　　　　　　　　 | 404  | 216ms　|
| 657  | ✅　　| 20.随机Fuzz　　　 | fuzz#7 PATCH /api/v1/assets/84662673-af90-4... → 200　　　　| 200  | 512ms　|
| 658  | ✅　　| 20.随机Fuzz　　　 | fuzz#8 DELETE /api/v1/assets/84662673-af90-4... → 200　　　 | 200  | 215ms　|
| 659  | ✅　　| 20.随机Fuzz　　　 | fuzz#9 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　　| 200  | 219ms　|
| 660  | ✅　　| 20.随机Fuzz　　　 | fuzz#10 DELETE /api/v1/assets/fuzz-794783 → 200　　　　　　 | 200  | 214ms　|
| 661  | ✅　　| 20.随机Fuzz　　　 | fuzz#11 PATCH /api/v1/assets/fuzz-880436 → 404　　　　　　　| 404  | 219ms　|
| 662  | ✅　　| 20.随机Fuzz　　　 | fuzz#12 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　 | 200  | 223ms　|
| 663  | ✅　　| 20.随机Fuzz　　　 | fuzz#13 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 216ms　|
| 664  | ✅　　| 20.随机Fuzz　　　 | fuzz#14 DELETE /api/v1/assets/fuzz-957843 → 200　　　　　　 | 200  | 220ms　|
| 665  | ✅　　| 20.随机Fuzz　　　 | fuzz#15 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 1ms　　|
| 666  | ✅　　| 20.随机Fuzz　　　 | fuzz#16 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 667  | ✅　　| 20.随机Fuzz　　　 | fuzz#17 DELETE /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　| 200  | 217ms　|
| 668  | ✅　　| 20.随机Fuzz　　　 | fuzz#18 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 669  | ✅　　| 20.随机Fuzz　　　 | fuzz#19 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　 | 200  | 219ms　|
| 670  | ✅　　| 20.随机Fuzz　　　 | fuzz#20 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　 | 200  | 449ms　|
| 671  | ✅　　| 20.随机Fuzz　　　 | fuzz#21 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 672  | ✅　　| 20.随机Fuzz　　　 | fuzz#22 DELETE /api/v1/assets/fuzz-660779 → 200　　　　　　 | 200  | 215ms　|
| 673  | ✅　　| 20.随机Fuzz　　　 | fuzz#23 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 450ms　|
| 674  | ✅　　| 20.随机Fuzz　　　 | fuzz#24 PATCH /api/v1/assets/fuzz-28180 → 404　　　　　　　 | 404  | 216ms　|
| 675  | ✅　　| 20.随机Fuzz　　　 | fuzz#25 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 676  | ✅　　| 20.随机Fuzz　　　 | fuzz#26 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 214ms　|
| 677  | ✅　　| 20.随机Fuzz　　　 | fuzz#27 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　 | 200  | 226ms　|
| 678  | ✅　　| 20.随机Fuzz　　　 | fuzz#28 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 444ms　|
| 679  | ✅　　| 20.随机Fuzz　　　 | fuzz#29 PATCH /api/v1/assets/fuzz-659975 → 404　　　　　　　| 404  | 216ms　|
| 680  | ✅　　| 20.随机Fuzz　　　 | fuzz#30 DELETE /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 216ms　|
| 681  | ✅　　| 20.随机Fuzz　　　 | fuzz#31 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　　| 200  | 217ms　|
| 682  | ✅　　| 20.随机Fuzz　　　 | fuzz#32 PATCH /api/v1/assets/fuzz-632251 → 404　　　　　　　| 404  | 214ms　|
| 683  | ✅　　| 20.随机Fuzz　　　 | fuzz#33 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 684  | ✅　　| 20.随机Fuzz　　　 | fuzz#34 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　 | 200  | 222ms　|
| 685  | ✅　　| 20.随机Fuzz　　　 | fuzz#35 GET /api/v1/assets/fuzz-716317 → 404　　　　　　　　| 404  | 228ms　|
| 686  | ✅　　| 20.随机Fuzz　　　 | fuzz#36 PATCH /api/v1/assets/fuzz-227011 → 404　　　　　　　| 404  | 221ms　|
| 687  | ✅　　| 20.随机Fuzz　　　 | fuzz#37 GET /api/v1/assets/fuzz-71287 → 404　　　　　　　　 | 404  | 420ms　|
| 688  | ✅　　| 20.随机Fuzz　　　 | fuzz#38 PATCH /api/v1/assets/fuzz-811103 → 404　　　　　　　| 404  | 215ms　|
| 689  | ✅　　| 20.随机Fuzz　　　 | fuzz#39 PATCH /api/v1/assets/fuzz-154892 → 404　　　　　　　| 404  | 217ms　|
| 690  | ✅　　| 20.随机Fuzz　　　 | fuzz#40 PATCH /api/v1/assets/fuzz-285189 → 404　　　　　　　| 404  | 216ms　|
| 691  | ✅　　| 20.随机Fuzz　　　 | fuzz#41 GET /api/v1/assets/fuzz-425096 → 404　　　　　　　　| 404  | 218ms　|
| 692  | ✅　　| 20.随机Fuzz　　　 | fuzz#42 PATCH /api/v1/assets/fuzz-830452 → 404　　　　　　　| 404  | 215ms　|
| 693  | ✅　　| 20.随机Fuzz　　　 | fuzz#43 DELETE /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 216ms　|
| 694  | ✅　　| 20.随机Fuzz　　　 | fuzz#44 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 436ms　|
| 695  | ✅　　| 20.随机Fuzz　　　 | fuzz#45 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 696  | ✅　　| 20.随机Fuzz　　　 | fuzz#46 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 437ms　|
| 697  | ✅　　| 20.随机Fuzz　　　 | fuzz#47 GET /api/v1/assets/fuzz-641620 → 404　　　　　　　　| 404  | 218ms　|
| 698  | ✅　　| 20.随机Fuzz　　　 | fuzz#48 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 699  | ✅　　| 20.随机Fuzz　　　 | fuzz#49 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 440ms　|
| 700  | ✅　　| 20.随机Fuzz　　　 | fuzz#50 PATCH /api/v1/assets/fuzz-974350 → 404　　　　　　　| 404  | 217ms　|
| 701  | ✅　　| 20.随机Fuzz　　　 | fuzz#51 DELETE /api/v1/assets/84662673-af90-4... → 200　　　| 200  | 221ms　|
| 702  | ✅　　| 20.随机Fuzz　　　 | fuzz#52 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　 | 200  | 223ms　|
| 703  | ✅　　| 20.随机Fuzz　　　 | fuzz#53 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 704  | ✅　　| 20.随机Fuzz　　　 | fuzz#54 PATCH /api/v1/assets/fuzz-671387 → 404　　　　　　　| 404  | 215ms　|
| 705  | ✅　　| 20.随机Fuzz　　　 | fuzz#55 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　 | 200  | 217ms　|
| 706  | ✅　　| 20.随机Fuzz　　　 | fuzz#56 DELETE /api/v1/assets/fuzz-361295 → 200　　　　　　 | 200  | 217ms　|
| 707  | ✅　　| 20.随机Fuzz　　　 | fuzz#57 DELETE /api/v1/assets/fuzz-92490 → 200　　　　　　　| 200  | 223ms　|
| 708  | ✅　　| 20.随机Fuzz　　　 | fuzz#58 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 709  | ✅　　| 20.随机Fuzz　　　 | fuzz#59 DELETE /api/v1/assets/fuzz-835602 → 200　　　　　　 | 200  | 217ms　|
| 710  | ✅　　| 20.随机Fuzz　　　 | fuzz#60 DELETE /api/v1/assets/fuzz-535349 → 200　　　　　　 | 200  | 218ms　|
| 711  | ✅　　| 20.随机Fuzz　　　 | fuzz#61 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 712  | ✅　　| 20.随机Fuzz　　　 | fuzz#62 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　 | 200  | 227ms　|
| 713  | ✅　　| 20.随机Fuzz　　　 | fuzz#63 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　 | 200  | 218ms　|
| 714  | ✅　　| 20.随机Fuzz　　　 | fuzz#64 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 438ms　|
| 715  | ✅　　| 20.随机Fuzz　　　 | fuzz#65 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 442ms　|
| 716  | ✅　　| 20.随机Fuzz　　　 | fuzz#66 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 450ms　|
| 717  | ✅　　| 20.随机Fuzz　　　 | fuzz#67 DELETE /api/v1/assets/84662673-af90-4... → 200　　　| 200  | 217ms　|
| 718  | ✅　　| 20.随机Fuzz　　　 | fuzz#68 PATCH /api/v1/assets/84662673-af90-4... → 200　　　 | 200  | 439ms　|
| 719  | ✅　　| 20.随机Fuzz　　　 | fuzz#69 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　 | 200  | 445ms　|
| 720  | ✅　　| 20.随机Fuzz　　　 | fuzz#70 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 214ms　|
| 721  | ✅　　| 20.随机Fuzz　　　 | fuzz#71 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 441ms　|
| 722  | ✅　　| 20.随机Fuzz　　　 | fuzz#72 PATCH /api/v1/assets/5057cbf1-dd05-4... → 200　　　 | 200  | 443ms　|
| 723  | ✅　　| 20.随机Fuzz　　　 | fuzz#73 GET /api/v1/assets/fuzz-406698 → 404　　　　　　　　| 404  | 216ms　|
| 724  | ✅　　| 20.随机Fuzz　　　 | fuzz#74 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 725  | ✅　　| 20.随机Fuzz　　　 | fuzz#75 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 445ms　|
| 726  | ✅　　| 20.随机Fuzz　　　 | fuzz#76 PATCH /api/v1/assets/5057cbf1-dd05-4... → 200　　　 | 200  | 450ms　|
| 727  | ✅　　| 20.随机Fuzz　　　 | fuzz#77 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　 | 200  | 217ms　|
| 728  | ✅　　| 20.随机Fuzz　　　 | fuzz#78 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 458ms　|
| 729  | ✅　　| 20.随机Fuzz　　　 | fuzz#79 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 444ms　|
| 730  | ✅　　| 20.随机Fuzz　　　 | fuzz#80 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 731  | ✅　　| 20.随机Fuzz　　　 | fuzz#81 GET /api/v1/assets/fuzz-647385 → 404　　　　　　　　| 404  | 217ms　|
| 732  | ✅　　| 20.随机Fuzz　　　 | fuzz#82 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 733  | ✅　　| 20.随机Fuzz　　　 | fuzz#83 GET /api/v1/assets/fuzz-247302 → 404　　　　　　　　| 404  | 215ms　|
| 734  | ✅　　| 20.随机Fuzz　　　 | fuzz#84 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 437ms　|
| 735  | ✅　　| 20.随机Fuzz　　　 | fuzz#85 GET /api/v1/assets/fuzz-420982 → 404　　　　　　　　| 404  | 216ms　|
| 736  | ✅　　| 20.随机Fuzz　　　 | fuzz#86 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 215ms　|
| 737  | ✅　　| 20.随机Fuzz　　　 | fuzz#87 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　 | 200  | 219ms　|
| 738  | ✅　　| 20.随机Fuzz　　　 | fuzz#88 DELETE /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　| 200  | 216ms　|
| 739  | ✅　　| 20.随机Fuzz　　　 | fuzz#89 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　 | 200  | 219ms　|
| 740  | ✅　　| 20.随机Fuzz　　　 | fuzz#90 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 444ms　|
| 741  | ✅　　| 20.随机Fuzz　　　 | fuzz#91 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 447ms　|
| 742  | ✅　　| 20.随机Fuzz　　　 | fuzz#92 DELETE /api/v1/assets/fuzz-603019 → 200　　　　　　 | 200  | 217ms　|
| 743  | ✅　　| 20.随机Fuzz　　　 | fuzz#93 DELETE /api/v1/assets/84662673-af90-4... → 200　　　| 200  | 216ms　|
| 744  | ✅　　| 20.随机Fuzz　　　 | fuzz#94 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 439ms　|
| 745  | ✅　　| 20.随机Fuzz　　　 | fuzz#95 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　 | 200  | 443ms　|
| 746  | ✅　　| 20.随机Fuzz　　　 | fuzz#96 POST /api/v1/assets → 201　　　　　　　　　　　　　 | 201  | 461ms　|
| 747  | ✅　　| 20.随机Fuzz　　　 | fuzz#97 PATCH /api/v1/assets/fuzz-306972 → 404　　　　　　　| 404  | 214ms　|
| 748  | ✅　　| 20.随机Fuzz　　　 | fuzz#98 DELETE /api/v1/assets/fuzz-914976 → 200　　　　　　 | 200  | 216ms　|
| 749  | ✅　　| 20.随机Fuzz　　　 | fuzz#99 GET /healthz → 200　　　　　　　　　　　　　　　　　| 200  | 0s　　 |
| 750  | ✅　　| 20.随机Fuzz　　　 | fuzz#100 GET /api/v1/assets/fuzz-964342 → 404　　　　　　　 | 404  | 216ms　|
| 751  | ✅　　| 20.随机Fuzz　　　 | fuzz#101 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 752  | ✅　　| 20.随机Fuzz　　　 | fuzz#102 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 753  | ✅　　| 20.随机Fuzz　　　 | fuzz#103 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 754  | ✅　　| 20.随机Fuzz　　　 | fuzz#104 PATCH /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　| 200  | 450ms　|
| 755  | ✅　　| 20.随机Fuzz　　　 | fuzz#105 DELETE /api/v1/assets/fuzz-687425 → 200　　　　　　| 200  | 217ms　|
| 756  | ✅　　| 20.随机Fuzz　　　 | fuzz#106 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　| 200  | 219ms　|
| 757  | ✅　　| 20.随机Fuzz　　　 | fuzz#107 DELETE /api/v1/assets/fa3e5c5e-0a25-4... → 200　　 | 200  | 221ms　|
| 758  | ✅　　| 20.随机Fuzz　　　 | fuzz#108 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 759  | ✅　　| 20.随机Fuzz　　　 | fuzz#109 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 443ms　|
| 760  | ✅　　| 20.随机Fuzz　　　 | fuzz#110 DELETE /api/v1/assets/fuzz-410033 → 200　　　　　　| 200  | 219ms　|
| 761  | ✅　　| 20.随机Fuzz　　　 | fuzz#111 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 762  | ✅　　| 20.随机Fuzz　　　 | fuzz#112 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　| 200  | 224ms　|
| 763  | ✅　　| 20.随机Fuzz　　　 | fuzz#113 DELETE /api/v1/assets/fuzz-797247 → 200　　　　　　| 200  | 218ms　|
| 764  | ✅　　| 20.随机Fuzz　　　 | fuzz#114 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 765  | ✅　　| 20.随机Fuzz　　　 | fuzz#115 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 441ms　|
| 766  | ✅　　| 20.随机Fuzz　　　 | fuzz#116 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 767  | ✅　　| 20.随机Fuzz　　　 | fuzz#117 GET /api/v1/assets/fuzz-235715 → 404　　　　　　　 | 404  | 218ms　|
| 768  | ✅　　| 20.随机Fuzz　　　 | fuzz#118 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 443ms　|
| 769  | ✅　　| 20.随机Fuzz　　　 | fuzz#119 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 770  | ✅　　| 20.随机Fuzz　　　 | fuzz#120 DELETE /api/v1/assets/fuzz-67449 → 200　　　　　　 | 200  | 221ms　|
| 771  | ✅　　| 20.随机Fuzz　　　 | fuzz#121 DELETE /api/v1/assets/fuzz-830164 → 200　　　　　　| 200  | 215ms　|
| 772  | ✅　　| 20.随机Fuzz　　　 | fuzz#122 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 553ms　|
| 773  | ✅　　| 20.随机Fuzz　　　 | fuzz#123 PATCH /api/v1/assets/fuzz-183129 → 404　　　　　　 | 404  | 216ms　|
| 774  | ✅　　| 20.随机Fuzz　　　 | fuzz#124 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 775  | ✅　　| 20.随机Fuzz　　　 | fuzz#125 GET /api/v1/assets/fuzz-921391 → 404　　　　　　　 | 404  | 215ms　|
| 776  | ✅　　| 20.随机Fuzz　　　 | fuzz#126 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 227ms　|
| 777  | ✅　　| 20.随机Fuzz　　　 | fuzz#127 GET /api/v1/assets/fuzz-31032 → 404　　　　　　　　| 404  | 214ms　|
| 778  | ✅　　| 20.随机Fuzz　　　 | fuzz#128 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 445ms　|
| 779  | ✅　　| 20.随机Fuzz　　　 | fuzz#129 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 780  | ✅　　| 20.随机Fuzz　　　 | fuzz#130 GET /api/v1/assets/fuzz-782461 → 404　　　　　　　 | 404  | 216ms　|
| 781  | ✅　　| 20.随机Fuzz　　　 | fuzz#131 DELETE /api/v1/assets/fuzz-928589 → 200　　　　　　| 200  | 216ms　|
| 782  | ✅　　| 20.随机Fuzz　　　 | fuzz#132 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　| 200  | 219ms　|
| 783  | ✅　　| 20.随机Fuzz　　　 | fuzz#133 PATCH /api/v1/assets/fuzz-744798 → 404　　　　　　 | 404  | 215ms　|
| 784  | ✅　　| 20.随机Fuzz　　　 | fuzz#134 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 435ms　|
| 785  | ✅　　| 20.随机Fuzz　　　 | fuzz#135 GET /api/v1/assets/84662673-af90-4... → 200　　　　| 200  | 217ms　|
| 786  | ✅　　| 20.随机Fuzz　　　 | fuzz#136 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 437ms　|
| 787  | ✅　　| 20.随机Fuzz　　　 | fuzz#137 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 437ms　|
| 788  | ✅　　| 20.随机Fuzz　　　 | fuzz#138 GET /api/v1/assets/fuzz-240697 → 404　　　　　　　 | 404  | 215ms　|
| 789  | ✅　　| 20.随机Fuzz　　　 | fuzz#139 DELETE /api/v1/assets/fuzz-22379 → 200　　　　　　 | 200  | 216ms　|
| 790  | ✅　　| 20.随机Fuzz　　　 | fuzz#140 PATCH /api/v1/assets/84662673-af90-4... → 200　　　| 200  | 440ms　|
| 791  | ✅　　| 20.随机Fuzz　　　 | fuzz#141 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 441ms　|
| 792  | ✅　　| 20.随机Fuzz　　　 | fuzz#142 PATCH /api/v1/assets/fuzz-894463 → 404　　　　　　 | 404  | 215ms　|
| 793  | ✅　　| 20.随机Fuzz　　　 | fuzz#143 PATCH /api/v1/assets/fuzz-395905 → 404　　　　　　 | 404  | 215ms　|
| 794  | ✅　　| 20.随机Fuzz　　　 | fuzz#144 DELETE /api/v1/assets/a2ebc37d-ef73-4... → 200　　 | 200  | 214ms　|
| 795  | ✅　　| 20.随机Fuzz　　　 | fuzz#145 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 796  | ✅　　| 20.随机Fuzz　　　 | fuzz#146 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 214ms　|
| 797  | ✅　　| 20.随机Fuzz　　　 | fuzz#147 PATCH /api/v1/assets/fuzz-734075 → 404　　　　　　 | 404  | 217ms　|
| 798  | ✅　　| 20.随机Fuzz　　　 | fuzz#148 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 799  | ✅　　| 20.随机Fuzz　　　 | fuzz#149 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 442ms　|
| 800  | ✅　　| 20.随机Fuzz　　　 | fuzz#150 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 801  | ✅　　| 20.随机Fuzz　　　 | fuzz#151 PATCH /api/v1/assets/fuzz-333846 → 404　　　　　　 | 404  | 216ms　|
| 802  | ✅　　| 20.随机Fuzz　　　 | fuzz#152 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 803  | ✅　　| 20.随机Fuzz　　　 | fuzz#153 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 804  | ✅　　| 20.随机Fuzz　　　 | fuzz#154 GET /api/v1/assets/fuzz-996675 → 404　　　　　　　 | 404  | 218ms　|
| 805  | ✅　　| 20.随机Fuzz　　　 | fuzz#155 DELETE /api/v1/assets/fa3e5c5e-0a25-4... → 200　　 | 200  | 218ms　|
| 806  | ✅　　| 20.随机Fuzz　　　 | fuzz#156 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 441ms　|
| 807  | ✅　　| 20.随机Fuzz　　　 | fuzz#157 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 808  | ✅　　| 20.随机Fuzz　　　 | fuzz#158 DELETE /api/v1/assets/fuzz-784342 → 200　　　　　　| 200  | 217ms　|
| 809  | ✅　　| 20.随机Fuzz　　　 | fuzz#159 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 217ms　|
| 810  | ✅　　| 20.随机Fuzz　　　 | fuzz#160 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 811  | ✅　　| 20.随机Fuzz　　　 | fuzz#161 DELETE /api/v1/assets/fuzz-487504 → 200　　　　　　| 200  | 217ms　|
| 812  | ✅　　| 20.随机Fuzz　　　 | fuzz#162 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 813  | ✅　　| 20.随机Fuzz　　　 | fuzz#163 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 442ms　|
| 814  | ✅　　| 20.随机Fuzz　　　 | fuzz#164 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　| 200  | 217ms　|
| 815  | ✅　　| 20.随机Fuzz　　　 | fuzz#165 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 816  | ✅　　| 20.随机Fuzz　　　 | fuzz#166 DELETE /api/v1/assets/fuzz-784201 → 200　　　　　　| 200  | 215ms　|
| 817  | ✅　　| 20.随机Fuzz　　　 | fuzz#167 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 818  | ✅　　| 20.随机Fuzz　　　 | fuzz#168 DELETE /api/v1/assets/fuzz-85274 → 200　　　　　　 | 200  | 220ms　|
| 819  | ✅　　| 20.随机Fuzz　　　 | fuzz#169 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 820  | ✅　　| 20.随机Fuzz　　　 | fuzz#170 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 821  | ✅　　| 20.随机Fuzz　　　 | fuzz#171 GET /api/v1/assets/84662673-af90-4... → 200　　　　| 200  | 217ms　|
| 822  | ✅　　| 20.随机Fuzz　　　 | fuzz#172 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 446ms　|
| 823  | ✅　　| 20.随机Fuzz　　　 | fuzz#173 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 824  | ✅　　| 20.随机Fuzz　　　 | fuzz#174 DELETE /api/v1/assets/fuzz-711839 → 200　　　　　　| 200  | 217ms　|
| 825  | ✅　　| 20.随机Fuzz　　　 | fuzz#175 DELETE /api/v1/assets/fa3e5c5e-0a25-4... → 200　　 | 200  | 215ms　|
| 826  | ✅　　| 20.随机Fuzz　　　 | fuzz#176 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 827  | ✅　　| 20.随机Fuzz　　　 | fuzz#177 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 828  | ✅　　| 20.随机Fuzz　　　 | fuzz#178 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 445ms　|
| 829  | ✅　　| 20.随机Fuzz　　　 | fuzz#179 PATCH /api/v1/assets/fuzz-506887 → 404　　　　　　 | 404  | 230ms　|
| 830  | ✅　　| 20.随机Fuzz　　　 | fuzz#180 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 1ms　　|
| 831  | ✅　　| 20.随机Fuzz　　　 | fuzz#181 GET /api/v1/assets/fuzz-997002 → 404　　　　　　　 | 404  | 217ms　|
| 832  | ✅　　| 20.随机Fuzz　　　 | fuzz#182 PATCH /api/v1/assets/fuzz-967446 → 404　　　　　　 | 404  | 218ms　|
| 833  | ✅　　| 20.随机Fuzz　　　 | fuzz#183 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 443ms　|
| 834  | ✅　　| 20.随机Fuzz　　　 | fuzz#184 DELETE /api/v1/assets/a2ebc37d-ef73-4... → 200　　 | 200  | 217ms　|
| 835  | ✅　　| 20.随机Fuzz　　　 | fuzz#185 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 836  | ✅　　| 20.随机Fuzz　　　 | fuzz#186 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 837  | ✅　　| 20.随机Fuzz　　　 | fuzz#187 DELETE /api/v1/assets/fuzz-227942 → 200　　　　　　| 200  | 218ms　|
| 838  | ✅　　| 20.随机Fuzz　　　 | fuzz#188 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 445ms　|
| 839  | ✅　　| 20.随机Fuzz　　　 | fuzz#189 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 840  | ✅　　| 20.随机Fuzz　　　 | fuzz#190 DELETE /api/v1/assets/fuzz-356049 → 200　　　　　　| 200  | 218ms　|
| 841  | ✅　　| 20.随机Fuzz　　　 | fuzz#191 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 442ms　|
| 842  | ✅　　| 20.随机Fuzz　　　 | fuzz#192 GET /api/v1/assets/fuzz-111041 → 404　　　　　　　 | 404  | 217ms　|
| 843  | ✅　　| 20.随机Fuzz　　　 | fuzz#193 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 844  | ✅　　| 20.随机Fuzz　　　 | fuzz#194 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 845  | ✅　　| 20.随机Fuzz　　　 | fuzz#195 PATCH /api/v1/assets/5057cbf1-dd05-4... → 200　　　| 200  | 443ms　|
| 846  | ✅　　| 20.随机Fuzz　　　 | fuzz#196 GET /api/v1/assets/fuzz-798366 → 404　　　　　　　 | 404  | 215ms　|
| 847  | ✅　　| 20.随机Fuzz　　　 | fuzz#197 DELETE /api/v1/assets/fuzz-335395 → 200　　　　　　| 200  | 215ms　|
| 848  | ✅　　| 20.随机Fuzz　　　 | fuzz#198 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 849  | ✅　　| 20.随机Fuzz　　　 | fuzz#199 PATCH /api/v1/assets/fuzz-719784 → 404　　　　　　 | 404  | 216ms　|
| 850  | ✅　　| 20.随机Fuzz　　　 | fuzz#200 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 851  | ✅　　| 20.随机Fuzz　　　 | fuzz#201 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 852  | ✅　　| 20.随机Fuzz　　　 | fuzz#202 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 853  | ✅　　| 20.随机Fuzz　　　 | fuzz#203 DELETE /api/v1/assets/fuzz-100956 → 200　　　　　　| 200  | 223ms　|
| 854  | ✅　　| 20.随机Fuzz　　　 | fuzz#204 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 855  | ✅　　| 20.随机Fuzz　　　 | fuzz#205 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 445ms　|
| 856  | ✅　　| 20.随机Fuzz　　　 | fuzz#206 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 857  | ✅　　| 20.随机Fuzz　　　 | fuzz#207 PATCH /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　| 200  | 441ms　|
| 858  | ✅　　| 20.随机Fuzz　　　 | fuzz#208 GET /api/v1/assets/fuzz-622501 → 404　　　　　　　 | 404  | 214ms　|
| 859  | ✅　　| 20.随机Fuzz　　　 | fuzz#209 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 215ms　|
| 860  | ✅　　| 20.随机Fuzz　　　 | fuzz#210 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 247ms　|
| 861  | ✅　　| 20.随机Fuzz　　　 | fuzz#211 DELETE /api/v1/assets/fuzz-800446 → 200　　　　　　| 200  | 219ms　|
| 862  | ✅　　| 20.随机Fuzz　　　 | fuzz#212 DELETE /api/v1/assets/fuzz-19633 → 200　　　　　　 | 200  | 218ms　|
| 863  | ✅　　| 20.随机Fuzz　　　 | fuzz#213 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 215ms　|
| 864  | ✅　　| 20.随机Fuzz　　　 | fuzz#214 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 865  | ✅　　| 20.随机Fuzz　　　 | fuzz#215 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 866  | ✅　　| 20.随机Fuzz　　　 | fuzz#216 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 867  | ✅　　| 20.随机Fuzz　　　 | fuzz#217 GET /api/v1/assets/fuzz-733519 → 404　　　　　　　 | 404  | 219ms　|
| 868  | ✅　　| 20.随机Fuzz　　　 | fuzz#218 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　 | 200  | 222ms　|
| 869  | ✅　　| 20.随机Fuzz　　　 | fuzz#219 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 870  | ✅　　| 20.随机Fuzz　　　 | fuzz#220 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 871  | ✅　　| 20.随机Fuzz　　　 | fuzz#221 PATCH /api/v1/assets/5057cbf1-dd05-4... → 200　　　| 200  | 441ms　|
| 872  | ✅　　| 20.随机Fuzz　　　 | fuzz#222 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 444ms　|
| 873  | ✅　　| 20.随机Fuzz　　　 | fuzz#223 DELETE /api/v1/assets/fa3e5c5e-0a25-4... → 200　　 | 200  | 216ms　|
| 874  | ✅　　| 20.随机Fuzz　　　 | fuzz#224 GET /api/v1/assets/fuzz-561861 → 404　　　　　　　 | 404  | 216ms　|
| 875  | ✅　　| 20.随机Fuzz　　　 | fuzz#225 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 876  | ✅　　| 20.随机Fuzz　　　 | fuzz#226 DELETE /api/v1/assets/fuzz-265210 → 200　　　　　　| 200  | 217ms　|
| 877  | ✅　　| 20.随机Fuzz　　　 | fuzz#227 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 878  | ✅　　| 20.随机Fuzz　　　 | fuzz#228 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 443ms　|
| 879  | ✅　　| 20.随机Fuzz　　　 | fuzz#229 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 880  | ✅　　| 20.随机Fuzz　　　 | fuzz#230 DELETE /api/v1/assets/fuzz-586994 → 200　　　　　　| 200  | 217ms　|
| 881  | ✅　　| 20.随机Fuzz　　　 | fuzz#231 PATCH /api/v1/assets/fuzz-970565 → 404　　　　　　 | 404  | 214ms　|
| 882  | ✅　　| 20.随机Fuzz　　　 | fuzz#232 GET /api/v1/assets/fuzz-482271 → 404　　　　　　　 | 404  | 218ms　|
| 883  | ✅　　| 20.随机Fuzz　　　 | fuzz#233 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 229ms　|
| 884  | ✅　　| 20.随机Fuzz　　　 | fuzz#234 PATCH /api/v1/assets/fuzz-303303 → 404　　　　　　 | 404  | 216ms　|
| 885  | ✅　　| 20.随机Fuzz　　　 | fuzz#235 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 273ms　|
| 886  | ✅　　| 20.随机Fuzz　　　 | fuzz#236 GET /api/v1/assets/fuzz-418039 → 404　　　　　　　 | 404  | 216ms　|
| 887  | ✅　　| 20.随机Fuzz　　　 | fuzz#237 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 438ms　|
| 888  | ✅　　| 20.随机Fuzz　　　 | fuzz#238 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　| 200  | 220ms　|
| 889  | ✅　　| 20.随机Fuzz　　　 | fuzz#239 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 890  | ✅　　| 20.随机Fuzz　　　 | fuzz#240 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　| 200  | 241ms　|
| 891  | ✅　　| 20.随机Fuzz　　　 | fuzz#241 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 892  | ✅　　| 20.随机Fuzz　　　 | fuzz#242 GET /api/v1/assets/84662673-af90-4... → 200　　　　| 200  | 218ms　|
| 893  | ✅　　| 20.随机Fuzz　　　 | fuzz#243 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 457ms　|
| 894  | ✅　　| 20.随机Fuzz　　　 | fuzz#244 GET /api/v1/assets/fuzz-405659 → 404　　　　　　　 | 404  | 215ms　|
| 895  | ✅　　| 20.随机Fuzz　　　 | fuzz#245 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 896  | ✅　　| 20.随机Fuzz　　　 | fuzz#246 DELETE /api/v1/assets/fuzz-192819 → 200　　　　　　| 200  | 217ms　|
| 897  | ✅　　| 20.随机Fuzz　　　 | fuzz#247 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　| 200  | 217ms　|
| 898  | ✅　　| 20.随机Fuzz　　　 | fuzz#248 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 458ms　|
| 899  | ✅　　| 20.随机Fuzz　　　 | fuzz#249 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 451ms　|
| 900  | ✅　　| 20.随机Fuzz　　　 | fuzz#250 DELETE /api/v1/assets/fuzz-942317 → 200　　　　　　| 200  | 230ms　|
| 901  | ✅　　| 20.随机Fuzz　　　 | fuzz#251 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　 | 200  | 216ms　|
| 902  | ✅　　| 20.随机Fuzz　　　 | fuzz#252 DELETE /api/v1/assets/fuzz-127671 → 200　　　　　　| 200  | 217ms　|
| 903  | ✅　　| 20.随机Fuzz　　　 | fuzz#253 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　| 200  | 220ms　|
| 904  | ✅　　| 20.随机Fuzz　　　 | fuzz#254 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 905  | ✅　　| 20.随机Fuzz　　　 | fuzz#255 DELETE /api/v1/assets/a2ebc37d-ef73-4... → 200　　 | 200  | 215ms　|
| 906  | ✅　　| 20.随机Fuzz　　　 | fuzz#256 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 907  | ✅　　| 20.随机Fuzz　　　 | fuzz#257 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 438ms　|
| 908  | ✅　　| 20.随机Fuzz　　　 | fuzz#258 GET /api/v1/assets/fuzz-243682 → 404　　　　　　　 | 404  | 279ms　|
| 909  | ✅　　| 20.随机Fuzz　　　 | fuzz#259 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 910  | ✅　　| 20.随机Fuzz　　　 | fuzz#260 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 911  | ✅　　| 20.随机Fuzz　　　 | fuzz#261 PATCH /api/v1/assets/fuzz-934712 → 404　　　　　　 | 404  | 216ms　|
| 912  | ✅　　| 20.随机Fuzz　　　 | fuzz#262 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　 | 200  | 215ms　|
| 913  | ✅　　| 20.随机Fuzz　　　 | fuzz#263 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 219ms　|
| 914  | ✅　　| 20.随机Fuzz　　　 | fuzz#264 PATCH /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　| 200  | 443ms　|
| 915  | ✅　　| 20.随机Fuzz　　　 | fuzz#265 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 448ms　|
| 916  | ✅　　| 20.随机Fuzz　　　 | fuzz#266 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 220ms　|
| 917  | ✅　　| 20.随机Fuzz　　　 | fuzz#267 GET /api/v1/assets/fuzz-439181 → 404　　　　　　　 | 404  | 214ms　|
| 918  | ✅　　| 20.随机Fuzz　　　 | fuzz#268 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 919  | ✅　　| 20.随机Fuzz　　　 | fuzz#269 DELETE /api/v1/assets/fuzz-223533 → 200　　　　　　| 200  | 216ms　|
| 920  | ✅　　| 20.随机Fuzz　　　 | fuzz#270 DELETE /api/v1/assets/84662673-af90-4... → 200　　 | 200  | 217ms　|
| 921  | ✅　　| 20.随机Fuzz　　　 | fuzz#271 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 922  | ✅　　| 20.随机Fuzz　　　 | fuzz#272 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 451ms　|
| 923  | ✅　　| 20.随机Fuzz　　　 | fuzz#273 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 924  | ✅　　| 20.随机Fuzz　　　 | fuzz#274 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 925  | ✅　　| 20.随机Fuzz　　　 | fuzz#275 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　| 200  | 221ms　|
| 926  | ✅　　| 20.随机Fuzz　　　 | fuzz#276 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　| 200  | 219ms　|
| 927  | ✅　　| 20.随机Fuzz　　　 | fuzz#277 DELETE /api/v1/assets/fuzz-246348 → 200　　　　　　| 200  | 221ms　|
| 928  | ✅　　| 20.随机Fuzz　　　 | fuzz#278 GET /api/v1/assets/fuzz-787975 → 404　　　　　　　 | 404  | 216ms　|
| 929  | ✅　　| 20.随机Fuzz　　　 | fuzz#279 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 930  | ✅　　| 20.随机Fuzz　　　 | fuzz#280 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 931  | ✅　　| 20.随机Fuzz　　　 | fuzz#281 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 932  | ✅　　| 20.随机Fuzz　　　 | fuzz#282 DELETE /api/v1/assets/fuzz-777588 → 200　　　　　　| 200  | 215ms　|
| 933  | ✅　　| 20.随机Fuzz　　　 | fuzz#283 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 440ms　|
| 934  | ✅　　| 20.随机Fuzz　　　 | fuzz#284 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 444ms　|
| 935  | ✅　　| 20.随机Fuzz　　　 | fuzz#285 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 936  | ✅　　| 20.随机Fuzz　　　 | fuzz#286 PATCH /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　| 200  | 440ms　|
| 937  | ✅　　| 20.随机Fuzz　　　 | fuzz#287 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 938  | ✅　　| 20.随机Fuzz　　　 | fuzz#288 GET /api/v1/assets/fuzz-984508 → 404　　　　　　　 | 404  | 217ms　|
| 939  | ✅　　| 20.随机Fuzz　　　 | fuzz#289 DELETE /api/v1/assets/fuzz-580423 → 200　　　　　　| 200  | 214ms　|
| 940  | ✅　　| 20.随机Fuzz　　　 | fuzz#290 DELETE /api/v1/assets/84662673-af90-4... → 200　　 | 200  | 217ms　|
| 941  | ✅　　| 20.随机Fuzz　　　 | fuzz#291 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 436ms　|
| 942  | ✅　　| 20.随机Fuzz　　　 | fuzz#292 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 943  | ✅　　| 20.随机Fuzz　　　 | fuzz#293 DELETE /api/v1/assets/fuzz-300763 → 200　　　　　　| 200  | 216ms　|
| 944  | ✅　　| 20.随机Fuzz　　　 | fuzz#294 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 443ms　|
| 945  | ✅　　| 20.随机Fuzz　　　 | fuzz#295 DELETE /api/v1/assets/fuzz-623737 → 200　　　　　　| 200  | 236ms　|
| 946  | ✅　　| 20.随机Fuzz　　　 | fuzz#296 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 445ms　|
| 947  | ✅　　| 20.随机Fuzz　　　 | fuzz#297 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 215ms　|
| 948  | ✅　　| 20.随机Fuzz　　　 | fuzz#298 GET /api/v1/assets/fuzz-771860 → 404　　　　　　　 | 404  | 215ms　|
| 949  | ✅　　| 20.随机Fuzz　　　 | fuzz#299 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 950  | ✅　　| 20.随机Fuzz　　　 | fuzz#300 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　| 200  | 220ms　|
| 951  | ✅　　| 20.随机Fuzz　　　 | fuzz#301 DELETE /api/v1/assets/84662673-af90-4... → 200　　 | 200  | 217ms　|
| 952  | ✅　　| 20.随机Fuzz　　　 | fuzz#302 PATCH /api/v1/assets/84662673-af90-4... → 200　　　| 200  | 460ms　|
| 953  | ✅　　| 20.随机Fuzz　　　 | fuzz#303 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 445ms　|
| 954  | ✅　　| 20.随机Fuzz　　　 | fuzz#304 GET /api/v1/assets/ab4b77fe-cdc3-4... → 200　　　　| 200  | 218ms　|
| 955  | ✅　　| 20.随机Fuzz　　　 | fuzz#305 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 457ms　|
| 956  | ✅　　| 20.随机Fuzz　　　 | fuzz#306 GET /api/v1/assets/84662673-af90-4... → 200　　　　| 200  | 220ms　|
| 957  | ✅　　| 20.随机Fuzz　　　 | fuzz#307 GET /api/v1/assets/fuzz-524317 → 404　　　　　　　 | 404  | 215ms　|
| 958  | ✅　　| 20.随机Fuzz　　　 | fuzz#308 PATCH /api/v1/assets/5057cbf1-dd05-4... → 200　　　| 200  | 457ms　|
| 959  | ✅　　| 20.随机Fuzz　　　 | fuzz#309 DELETE /api/v1/assets/a2ebc37d-ef73-4... → 200　　 | 200  | 236ms　|
| 960  | ✅　　| 20.随机Fuzz　　　 | fuzz#310 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 439ms　|
| 961  | ✅　　| 20.随机Fuzz　　　 | fuzz#311 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 962  | ✅　　| 20.随机Fuzz　　　 | fuzz#312 DELETE /api/v1/assets/fuzz-487716 → 200　　　　　　| 200  | 215ms　|
| 963  | ✅　　| 20.随机Fuzz　　　 | fuzz#313 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 446ms　|
| 964  | ✅　　| 20.随机Fuzz　　　 | fuzz#314 GET /api/v1/assets/fuzz-753364 → 404　　　　　　　 | 404  | 219ms　|
| 965  | ✅　　| 20.随机Fuzz　　　 | fuzz#315 GET /api/v1/assets/fuzz-321515 → 404　　　　　　　 | 404  | 213ms　|
| 966  | ✅　　| 20.随机Fuzz　　　 | fuzz#316 GET /api/v1/assets/fuzz-258543 → 404　　　　　　　 | 404  | 218ms　|
| 967  | ✅　　| 20.随机Fuzz　　　 | fuzz#317 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 432ms　|
| 968  | ✅　　| 20.随机Fuzz　　　 | fuzz#318 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　 | 200  | 216ms　|
| 969  | ✅　　| 20.随机Fuzz　　　 | fuzz#319 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　| 200  | 212ms　|
| 970  | ✅　　| 20.随机Fuzz　　　 | fuzz#320 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 423ms　|
| 971  | ✅　　| 20.随机Fuzz　　　 | fuzz#321 GET /api/v1/assets/fuzz-292894 → 404　　　　　　　 | 404  | 210ms　|
| 972  | ✅　　| 20.随机Fuzz　　　 | fuzz#322 GET /api/v1/assets/fuzz-664002 → 404　　　　　　　 | 404  | 212ms　|
| 973  | ✅　　| 20.随机Fuzz　　　 | fuzz#323 DELETE /api/v1/assets/ab4b77fe-cdc3-4... → 200　　 | 200  | 213ms　|
| 974  | ✅　　| 20.随机Fuzz　　　 | fuzz#324 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 975  | ✅　　| 20.随机Fuzz　　　 | fuzz#325 PATCH /api/v1/assets/a2ebc37d-ef73-4... → 200　　　| 200  | 425ms　|
| 976  | ✅　　| 20.随机Fuzz　　　 | fuzz#326 PATCH /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　| 200  | 433ms　|
| 977  | ✅　　| 20.随机Fuzz　　　 | fuzz#327 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 426ms　|
| 978  | ✅　　| 20.随机Fuzz　　　 | fuzz#328 GET /api/v1/assets/fuzz-762206 → 404　　　　　　　 | 404  | 209ms　|
| 979  | ✅　　| 20.随机Fuzz　　　 | fuzz#329 GET /api/v1/assets/5057cbf1-dd05-4... → 200　　　　| 200  | 215ms　|
| 980  | ✅　　| 20.随机Fuzz　　　 | fuzz#330 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 435ms　|
| 981  | ✅　　| 20.随机Fuzz　　　 | fuzz#331 GET /api/v1/assets/fa3e5c5e-0a25-4... → 200　　　　| 200  | 212ms　|
| 982  | ✅　　| 20.随机Fuzz　　　 | fuzz#332 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　 | 200  | 210ms　|
| 983  | ✅　　| 20.随机Fuzz　　　 | fuzz#333 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 984  | ✅　　| 20.随机Fuzz　　　 | fuzz#334 DELETE /api/v1/assets/fuzz-543130 → 200　　　　　　| 200  | 210ms　|
| 985  | ✅　　| 20.随机Fuzz　　　 | fuzz#335 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 430ms　|
| 986  | ✅　　| 20.随机Fuzz　　　 | fuzz#336 DELETE /api/v1/assets/5057cbf1-dd05-4... → 200　　 | 200  | 213ms　|
| 987  | ✅　　| 20.随机Fuzz　　　 | fuzz#337 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 988  | ✅　　| 20.随机Fuzz　　　 | fuzz#338 PATCH /api/v1/assets/fuzz-825286 → 404　　　　　　 | 404  | 213ms　|
| 989  | ✅　　| 20.随机Fuzz　　　 | fuzz#339 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 990  | ✅　　| 20.随机Fuzz　　　 | fuzz#340 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 991  | ✅　　| 20.随机Fuzz　　　 | fuzz#341 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 419ms　|
| 992  | ✅　　| 20.随机Fuzz　　　 | fuzz#342 PATCH /api/v1/assets/fuzz-850946 → 404　　　　　　 | 404  | 211ms　|
| 993  | ✅　　| 20.随机Fuzz　　　 | fuzz#343 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 994  | ✅　　| 20.随机Fuzz　　　 | fuzz#344 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 424ms　|
| 995  | ✅　　| 20.随机Fuzz　　　 | fuzz#345 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 427ms　|
| 996  | ✅　　| 20.随机Fuzz　　　 | fuzz#346 DELETE /api/v1/assets/fuzz-721238 → 200　　　　　　| 200  | 210ms　|
| 997  | ✅　　| 20.随机Fuzz　　　 | fuzz#347 GET /api/v1/assets/a2ebc37d-ef73-4... → 200　　　　| 200  | 214ms　|
| 998  | ✅　　| 20.随机Fuzz　　　 | fuzz#348 POST /api/v1/assets → 201　　　　　　　　　　　　　| 201  | 424ms　|
| 999  | ✅　　| 20.随机Fuzz　　　 | fuzz#349 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |
| 1000 | ✅　　| 20.随机Fuzz　　　 | fuzz#350 GET /healthz → 200　　　　　　　　　　　　　　　　 | 200  | 0s　　 |

</details>
