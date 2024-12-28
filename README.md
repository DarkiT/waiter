# VPN Package

一个基于 Go 语言实现的 VPN 网络包，提供跨平台的虚拟网卡接口和数据转发功能。

## 包结构

```
waiter/
├── netlink/           # 网络接口操作包
│   ├── addr.go       # 网络地址相关定义
│   ├── link.go       # 网络接口相关定义
│   └── route.go      # 路由表相关定义
├── nic/              # 网络接口控制
│   ├── gvisor/       # 基于 gVisor 的网络栈实现
│   │   ├── forward.go    # 数据转发实现
│   │   ├── gvisor.go     # gVisor 虚拟网卡核心实现
│   │   ├── network.go    # 网络功能实现
│   │   ├── ping.go       # ICMP 实现
│   │   └── udp.go        # UDP 协议实现
│   ├── tun/          # 基于 TUN 的网络接口实现
│   │   ├── tun.go        # TUN 设备核心实现
│   │   └── tun_unix.go   # Unix 系统 TUN 实现
├── lru.go            # LRU 缓存实现
├── waiter.go         # 网络接口通用定义
├── packet.go         # IP 数据包处理
└── go.mod            # 项目依赖
```

## 虚拟网卡实现

### 1. gVisor 实现 (nic/gvisor)

基于 gVisor 用户态网络栈实现的虚拟网卡：

- **核心功能** (`gvisor.go`)
  - 完整的 TCP/IP 协议栈实现
  - IPv4/IPv6 双栈支持
  - 可配置的网络参数
  - MAC 地址管理

- **数据转发** (`forward.go`)
  - 高性能零拷贝数据转发
  - 支持多连接并发
  - 内置连接池优化
  - 自动超时清理

- **网络协议** (`network.go`, `ping.go`, `udp.go`)
  - TCP/UDP 协议支持
  - ICMP echo 实现
  - DNS 查询与解析
  - 连接管理与复用

### 2. TUN 实现 (nic/tun)

基于操作系统 TUN 设备实现的虚拟网卡：

- **跨平台支持** (`tun.go`, `tun_unix.go`)
  - Linux/macOS/Windows 支持
  - 原生 TUN 设备操作
  - 高性能数据包处理

- **主要特性**
  - 原生系统调用
  - 批量读写优化
  - MTU 自动配置
  - 地址自动管理

## 数据包处理

### Packet 结构
提供高效的 IP 数据包处理机制：

```go
type Packet struct {
    buf    []byte    // 数据包缓冲区
    offset int       // 数据偏移量
}

// 从包池获取数据包
packet := IPPacketPool.Get()
defer IPPacketPool.Put(packet)

// 写入数据
packet.Write(data)

// 获取 IP 版本
version := packet.Ver()

// 获取数据包内容
bytes := packet.AsBytes()

```

## 接口定义

### 通用网卡接口

```go
type NIC interface {
    // 读取 IP 数据包
    Read() (*Packet, error)
    
    // 写入 IP 数据包
    Write(*Packet) error
    
    // 关闭网卡
    Close() error
}
```

## 使用示例

### 1. 使用 gVisor 虚拟网卡

```go
// 创建 gVisor 虚拟网卡
config := nic.Config{
    Name: "gvisor0",
    MTU:  1500,
    IPv4: "192.168.1.1/24",
    IPv6: "fd00::1/64",
}
gvisorNIC, err := gvisor.Create(config)
if err != nil {
    log.Fatal(err)
}
defer gvisorNIC.Close()
```

### 2. 使用 TUN 虚拟网卡

```go
// 创建 TUN 设备
config := nic.Config{
    Name: "tun0",
    MTU:  1500,
    IPv4: "10.0.0.1/24",
}
tunNIC, err := tun.Create(config)
if err != nil {
    log.Fatal(err)
}
defer tunNIC.Close()
```

## 性能特性

1. gVisor 实现
- 用户态协议栈，避免内核切换
- 零拷贝数据转发
- 连接池复用
- 并发处理优化

2. TUN 实现
- 批量读写支持
- 系统调用优化
- 内存复用
- 高效数据包处理

## 注意事项

1. 权限要求
- gVisor 实现需要适当的系统权限
- TUN 实现需要 root/管理员权限

2. 平台差异
- TUN 实现在不同平台上的行为可能有差异
- Windows 平台加载TUN驱动需要管理员权限

3. 性能考虑
- gVisor 适合需要完整协议栈控制的场景
- TUN 适合需要高性能原生网络访问的场景

## 许可证

MIT License
