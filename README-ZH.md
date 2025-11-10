# MetaID Script Decoder

英文README: [README.md](README.md)

MetaID协议的纯Go语言解析工具库，用于从区块链交易中提取和解析PIN（Personal Information Node）数据。

## 支持的链

| 链名称 | 标识符 | 支持的格式 |
|--------|--------|------------|
| Bitcoin | `btc`, `bitcoin` | Witness (OP_FALSE + OP_IF), OP_RETURN |
| MicroVisionChain | `mvc`, `microvisionchain` | OP_RETURN |


## 快速开始

### 基础用法 - BTC链

```go
package main

import (
    "encoding/hex"
    "fmt"
    "log"

    "github.com/btcsuite/btcd/chaincfg"
    "github.com/metaid-developers/metaid-script-decoder/decoder/btc"
)

func main() {
    // 交易的十六进制字符串
    txHex := "your_transaction_hex_here"
    txBytes, _ := hex.DecodeString(txHex)

    // 创建BTC解析器
    parser := btc.NewBTCParser(nil)

    // 解析交易
    pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
    if err != nil {
        log.Fatal(err)
    }

    // 输出结果
    for _, pin := range pins {
        fmt.Printf("操作: %s, 路径: %s\n", pin.Operation, pin.Path)
        fmt.Printf("内容: %s\n", string(pin.ContentBody))
    }
}
```

### 基础用法 - MVC链

```go
import (
    "github.com/metaid-developers/metaid-script-decoder/decoder/mvc"
)

// 创建MVC解析器
parser := mvc.NewMVCParser(nil)

// 解析交易
pins, err := parser.ParseTransaction(txBytes, &chaincfg.MainNetParams)
```

### 使用自定义协议ID

```go
import (
    "github.com/metaid-developers/metaid-script-decoder/decoder"
    "github.com/metaid-developers/metaid-script-decoder/decoder/btc"
)

// 创建自定义配置
config := &decoder.ParserConfig{
    ProtocolID: "6d6574616964", // metaid的十六进制
}

// 使用自定义配置创建解析器
parser := btc.NewBTCParser(config)

// 解析交易
pins, err := parser.ParseTransaction(txBytes, &chaincfg.TestNet3Params)
```

## PIN数据结构

```go
type Pin struct {
    // 基础字段
    Operation  string // 操作类型: create, modify, revoke
    Path       string // PIN路径
    ParentPath string // 父路径

    // 内容字段
    ContentType   string // 内容类型
    ContentBody   []byte // 内容主体
    ContentLength uint64 // 内容长度

    // 元数据
    Encryption string // 加密方式
    Version    string // 版本

    // 区块链相关字段
    TxID    string // 交易ID
    Vout    uint32 // 输出索引
    Address string // 地址（PIN所有者）

    // 解析元数据
    ChainName string // 链名称: btc, mvc
    TxIndex   int    // 在交易中的索引位置
}
```

## MetaID协议说明

MetaID协议定义了一种在区块链上存储个人信息的标准格式。PIN（Personal Information Node）是协议的核心数据结构。

### PIN格式

#### Witness格式 (BTC)
```
OP_FALSE OP_IF <protocol_id> <operation> <path> <encryption> <version> <content_type> <payload> OP_ENDIF
```

#### OP_RETURN格式 (BTC/MVC)
```
OP_RETURN <protocol_id> <operation> <path> <encryption> <version> <content_type> <payload>
```

### 字段说明

- **protocol_id**: 协议标识符（默认：`6d6574616964` = "metaid"）
- **operation**: 操作类型
  - `create`: 创建新PIN
  - `modify`: 修改PIN
  - `revoke`: 撤销PIN
- **path**: PIN的路径，如 `/protocols/simplebuzz`
- **encryption**: 加密方式（默认：`0` = 未加密）
- **version**: 版本号（默认：`0`）
- **content_type**: 内容类型（默认：`application/json`）
- **payload**: 内容主体（可能分多个字段）

## 扩展支持新链

要添加对新区块链的支持，只需实现 `ChainParser` 接口：

```go
type ChainParser interface {
    ParseTransaction(txBytes []byte, chainParams interface{}) ([]*Pin, error)
    GetChainName() string
}
```

示例：

```go
package mychain

import "github.com/metaid-developers/metaid-script-decoder/decoder"

type MyChainParser struct {
    config *decoder.ParserConfig
}

func NewMyChainParser(config *decoder.ParserConfig) *MyChainParser {
    return &MyChainParser{config: config}
}

func (p *MyChainParser) GetChainName() string {
    return "mychain"
}

func (p *MyChainParser) ParseTransaction(txBytes []byte, chainParams interface{}) ([]*decoder.Pin, error) {
    // 实现解析逻辑
    return pins, nil
}
```

## 许可证

本项目采用与原项目相同的许可证。详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交Issue和Pull Request！

## 相关链接

- [MetaID协议文档](https://metaid.io)

