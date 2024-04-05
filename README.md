# Gitlab 代码检视意见导出工具
> 导出当天非closed状态MR的检视意见  
检视意见格式：【规范|功能】【提示|一般|严重】检视意见xxx  
检视意见示例：【规范】【提示】当前变量命名遵循驼峰命名规范

## Build
```bash
git clone https://github.com/firstep/codereview.git
cd codereview
go get
go build .
./codereview

# 执行导出任务一次,并设置抓取MR(更新时间)的开始时间
# ./codereview -t 20231125000000

```