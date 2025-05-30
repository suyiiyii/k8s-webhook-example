# Kubernetes ValidatingAdmissionPolicy 示例

## 概述

此项目展示了如何使用 Kubernetes 的 ValidatingAdmissionPolicy 特性来验证资源名称，防止资源名中包含特定关键词("suyiiyii")。

## 先决条件

- Kubernetes 1.26+ 集群（ValidatingAdmissionPolicy 在 1.26 中为 Beta 版）
- kubectl 命令行工具

## 部署方法

1. 应用 ValidatingAdmissionPolicy 和绑定:

```bash
kubectl apply -f validating-policy.yaml
```

2. 验证部署:

```bash
kubectl get validatingadmissionpolicy
kubectl get validatingadmissionpolicybinding
```

## 测试验证逻辑

创建一个包含禁止关键词的资源:

```bash
# 这将被拒绝
kubectl create namespace suyiiyii-test

# 这将被允许
kubectl create namespace allowed-test
```

## 与 Webhook 实现的区别

与基于 webhook 的实现相比，ValidatingAdmissionPolicy:

1. 无需维护额外的服务和证书
2. 使用 CEL 表达式直接在 Kubernetes API 服务器中执行
3. 具有更低的延迟和更高的可靠性
4. 更容易配置和管理
