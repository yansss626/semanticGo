# SemanticGo



## 1. 项目介绍

`SemanticGo` 是一个面向大语言模型应用场景的 AI 助手系统。

系统采用微服务架构设计，基于 `Go` 语言实现后端服务，支持流失和非流失输出，通过语义缓存，敏感词过滤，上下文管理以及 `OpenAI API`代理，实现一个具备高性能、低延迟和可扩展能力的智能对话系统。



### 1.1 产品 Logo

[![Semantic-Go-avatar-8d6aa6cf.jpg](https://i.postimg.cc/XY7pck5Z/Semantic-Go-avatar-8d6aa6cf.jpg)](https://postimg.cc/Lnw9mfNR)

### 1.2 界面布局

[![9f6783fc-282e-42f3-b9a6-500c7d53ff62.png](https://i.postimg.cc/d3dPMpyv/9f6783fc-282e-42f3-b9a6-500c7d53ff62.png)](https://postimg.cc/PvtFY6yF)

## 2. 系统架构图

[![Screenshot-2026-09-23-204115.png](https://i.postimg.cc/zv35NRSw/Screenshot-2026-09-23-204115.png)](https://postimg.cc/V0y2Bvh5)



## 3. 项目目录结构

```text	
semanticGo
├── ai-chat-web
│   └── Vue3 + Vite 前端
│
├── ai-chat-backend
│   └── 网关层
│
├── ai-chat-service
│   └── 业务服务层
│
├── keywords-filter
│   └── 敏感词过滤
│
├── reprise
│   └── 语义缓存
│
├── openai-api-proxy
│   └── 大语言模型 API 代理
│   
├── ai-chat-stack
│   └── Docker Stack 部署配置
│
└── tokenizer
│    └── token 计算服务
│   
└── README.md

```

## 4. 核心模块/功能

### 4.1 多轮对话（记忆功能）

基于 `Redis` 实现上下文管理，通过保存消息之间的父子节点关系（PID）实现多轮对话状态追踪。

用户每次发送消息时，系统会根据当前消息的 PID 获取历史对话记录，并将相关上下文信息传递给大语言模型，实现连续多轮对话。

可在配置文件 `ai-chat-service/dev_config.yaml` 中，自由配置  `context_len`，`context_ttl`，实现更长更持久的记忆对话轮次。

注：

需要合理配置 `max_tokens`与  `context_len`，避免历史上下文累积导致请求超过模型最大上下文窗口限制。



### 4.2 语义缓存

SemanticGo 支持基于向量检索的语义缓存机制，通过将用户 `Query` 转化为向量表示，搜索缓存，实现相似问题的缓存复用，减少大语言模型调用次数，降低响应延迟以及 API 调用成本。

当前语义缓存主要由以下模块组成：

- `Embedding` 模型：将用户 Query 转换为数字向量。可在 `ai-chat-service/dev_config.yaml` 中配置参数。
- `HNSW`索引：基于向量相似度（余弦相似度）进行近似最近邻搜索，快速检索相似历史问题，返回 K 个近邻向量。
- `Rerank`模型：对 HNSW 返回的 Top-K 候选 Query 进行重新排序，提高匹配准确性。可在 `ai-chat-service/dev_config.yaml` 中配置参数。
- `Cache`：保存历史 `Query`、对应向量以及模型回答。自定义 `Cache`（`Lidis`），实现精确查询，删除和修改。



#### 缓存建立流程

用户  `Query` 输入 `Embedding`  模型生成对应向量，并同步更新`HNSW` 索引。

同时将   `Query` 作为 `Key`，将对应的 `Vector` 和 `Answer` 作为 `Value` 保存至缓存中。

```text
Query
  |
  ↓
Embedding Model
  |
  ↓
Vector
  |
  ├──> HNSW Index
  |
  └──> Cache SET

Cache:
Key   : Query
Value : Vector + Answer
```

#### 缓存查询流程

用户  `Query` 输入 `Embedding`  模型生成对应向量，并通过 `HNSW` 索引搜索 `TopK`个相似历史   `Query`。

随后使用 `Rerank` 模型对候选结果进行精排，选择匹配度最高的  `Query` ，从缓存中获取历史回答。



```text
User Query
    |
    ↓
Embedding Model (生成 Query Vector )
    |   
    ↓ 
HNSW Search （返回 Top-K Similar Queries）
    |
    ↓
Rerank Model
    |
    ↓
Best Match Query
    |
    ↓
Cache GET
    |
    ↓
Cached Answer

```

语义缓存命中后，系统无需再次请求大语言模型，直接返回缓存结果。



为了方便用户感知当前回答来源，前端界面会显示当前回答类型：

- 大语言模型生成：标注公有大模型以及 `token` 消耗情况
- 缓存命中：显示节省 `token`数

示例：

大模型返回：

​	公有大模型 · 消耗 897 tokens

缓存命中：

​	缓存命中 · 节省 897 tokens

注： 

1. `Embedding` 模型可在配置文件 `reprise/dev_config.yaml` 中选择第三方模型参数，需要填入密钥，URL，以及模型等参数，也可以在本地部署。
2. `Rerank` 模型可在配置文件 `reprise/dev_config.yaml `中选择第三方模型参数需要填入密钥，URL，以及模型等参数，也可以在本地部署。
3. `HNSW` 算法基于开源项目`"github.com/fogfish/hnsw"`实现。
4. 目前语义缓存基于单轮  `Query`  进行语义匹配，属于无状态语义缓存。
5. 实现完整逻辑位于  `reprise`文件夹下。



### 4.3 敏感词检测

基于词库文件的敏感词检测，由服务层调用，检测文本是否与词库文件包含的敏感词匹配。

请求进入 AI 服务后，首先进行敏感词检测：

- 若检测到敏感词，则直接返回拦截结果；
- 若未检测到敏感词，则继续进入缓存查询以及大语言模型请求流程。

敏感词检测服务独立运行，通过 `gRPC` 与 服务层进行通信。

注：

敏感词检测算法基于`github.com/importcjj/sensitive`开源项目。



### 4.4 流式/非流式输出

支持流式和非流式两种响应模式。 

流式模式下，模型生成内容会被实时返回，用户无需等待完整回答生成，提高交互体验。

非流式模式下，一次性返回完整响应结果。



## 5. 部署/启动方式

支持单服务启动和 `Docker` 容器化部署两种方式。



### 5.1 单服务启动：

各服务可根据实际需求独立启动。



#### ai-chat-backend

```bash
cd ./ai-chat-backend
go run cmd/main.go
```

#### ai-chat-service

```bash
cd ./ai-chat-service
go run chat-server/main.go
```

#### keywords-filter

```bash
cd ./keywords-filter
go run filter-server/main,go --config=dev.config.yaml --dict=dict.txt
```

#### tokenizer

```bash
cd ./tokenizer
nuxt --port ${PORT} --module deepseek_tokenizer.py --workers 2
```

#### openai-api-proxy

```bash
cd ./openai-api-proxy
go run main.go
```



### 5.2 Docker 容器化部署

提供 `ai-chat-stack`部署配置，可通过 `docker stack `将多个服务统一部署。

```bash
cd ./ai-chat-stack
docker stack deploy -c compose.yaml ai-chat --resolve never
```

注：

1. 无论是单点启动还是容器化部署，都需要提前配置各服务对应的 IP 地址、端口以及相关模型参数。
2. `tokenizer` 需要根据不同的模型自行设置，不同模型的 `token` 计算方式可能存在差异。
3. 容器化部署需要提前在本地下载好镜像，并保证环境正常运行。

