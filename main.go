package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const KEYWORD = "suyiiyii"

var log = logrus.New()
var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	corev1.AddToScheme(scheme)
}

// validateResource 检查资源名称是否包含禁止的关键词
func validateResource(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	resp := &admissionv1.AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
	}

	// 从请求中获取资源名称
	resourceName := req.Name

	// 如果名称为空，尝试从对象中获取元数据
	if resourceName == "" {
		var metadata struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		}

		if err := json.Unmarshal(req.Object.Raw, &metadata); err == nil {
			resourceName = metadata.Metadata.Name
		}
	}

	// 检查资源名称是否包含关键词 "KEYWORD"
	if strings.Contains(strings.ToLower(resourceName), KEYWORD) {
		resp.Allowed = false
		resp.Result = &metav1.Status{
			Code:    403,
			Message: fmt.Sprintf("资源名称中不允许包含 '%s'，当前名称: %s", KEYWORD, resourceName),
		}

		log.WithFields(logrus.Fields{
			"resource": resourceName,
			"keyword":  KEYWORD,
			"allowed":  resp.Allowed,
		}).Warn("Resource validation failed")
	} else {
		log.WithFields(logrus.Fields{
			"resource": resourceName,
			"allowed":  resp.Allowed,
		}).Info("Resource validation passed")
	}

	return resp
}

// mutateResource 为 Pod 添加 create-by 标签
func mutateResource(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	resp := &admissionv1.AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
	}

	// 只处理 Pod 资源
	if req.Kind.Kind != "Pod" {
		return resp
	}

	// 解析 Pod 对象
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.WithError(err).Error("Failed to unmarshal pod object")
		resp.Allowed = false
		resp.Result = &metav1.Status{
			Code:    400,
			Message: fmt.Sprintf("Failed to unmarshal pod object: %v", err),
		}
		return resp
	}

	// 创建 JSON Patch 来添加标签
	patches := []map[string]interface{}{}

	// 如果 labels 不存在，先创建 labels 字段
	if pod.Labels == nil {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/metadata/labels",
			"value": map[string]string{},
		})
	}

	// 添加 create-by 标签
	patches = append(patches, map[string]interface{}{
		"op":    "add",
		"path":  "/metadata/labels/create-by",
		"value": "suyiiyii",
	})

	// 将 patches 序列化为 JSON
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		log.WithError(err).Error("Failed to marshal patches")
		resp.Allowed = false
		resp.Result = &metav1.Status{
			Code:    500,
			Message: fmt.Sprintf("Failed to marshal patches: %v", err),
		}
		return resp
	}

	// 设置 patch 类型和内容
	patchType := admissionv1.PatchTypeJSONPatch
	resp.Patch = patchBytes
	resp.PatchType = &patchType

	log.WithFields(logrus.Fields{
		"pod":   pod.Name,
		"patch": string(patchBytes),
	}).Info("Added create-by label to pod")

	return resp
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		log.WithError(err).Error("Failed to read request body")
		return
	}

	// 记录接收到的请求
	log.WithFields(logrus.Fields{
		"request": string(body),
	}).Info("Received admission review request")

	// 解析 AdmissionReview 请求
	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, "Failed to unmarshal request", http.StatusBadRequest)
		log.WithError(err).Error("Failed to unmarshal request")
		return
	}

	// 验证资源
	admissionResponse := validateResource(admissionReview.Request)

	// 构造响应
	admissionReview.Response = admissionResponse

	// 发送响应
	responseJSON, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		log.WithError(err).Error("Failed to marshal response")
		return
	}

	// 记录响应
	log.WithFields(logrus.Fields{
		"response": string(responseJSON),
		"allowed":  admissionResponse.Allowed,
	}).Info("Sending admission review response")

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		log.WithError(err).Error("Failed to read request body")
		return
	}

	// 记录接收到的请求
	log.WithFields(logrus.Fields{
		"request": string(body),
	}).Info("Received mutating admission review request")

	// 解析 AdmissionReview 请求
	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, "Failed to unmarshal request", http.StatusBadRequest)
		log.WithError(err).Error("Failed to unmarshal request")
		return
	}

	// 修改资源
	admissionResponse := mutateResource(admissionReview.Request)

	// 构造响应
	admissionReview.Response = admissionResponse

	// 发送响应
	responseJSON, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		log.WithError(err).Error("Failed to marshal response")
		return
	}

	// 记录响应
	log.WithFields(logrus.Fields{
		"response": string(responseJSON),
		"allowed":  admissionResponse.Allowed,
	}).Info("Sending mutating admission review response")

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func main() {
	// 配置 logrus
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	// 注册 webhook 处理器
	http.HandleFunc("/validate", handleValidate)
	http.HandleFunc("/mutate", handleMutate)

	// 启动服务器
	log.Info("Starting webhook server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.WithError(err).Fatal("Failed to start server")
	}
}
