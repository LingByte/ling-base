# toolkits

专项领域工具库集合。与 `common/`（应用内通用基建）和 `providers/`（第三方厂商适配）不同，这里放的是**可独立使用的主动工具 / 垂直领域库**，通常不应作为业务服务的默认依赖。

## 当前模块

| 模块 | 说明 |
|------|------|
| [`pentest`](./pentest) | Web 主动安全测试 / 扫描工具集 |

## 约定

1. **一子目录一 module**：路径形如 `github.com/LingByte/ling-base/toolkits/<name>`
2. **不进 full-mode 脚手架默认拷贝**（见 `lingcli/fullmode.go` 对 `toolkits` 的排除）
3. **使用边界写进各模块 README**（授权范围、误报启发式、依赖说明）
4. 新增同类库时直接放在 `toolkits/` 下，并注册到根目录 `go.work`
