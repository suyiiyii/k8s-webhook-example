package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const KEYWORD = "SUYIIYII"

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
	if strings.Contains(strings.ToUpper(resourceName), KEYWORD) {
		resp.Allowed = false
		resp.Result = &metav1.Status{
			Code:    403,
			Message: fmt.Sprintf("资源名称中不允许包含 '%s'，当前名称: %s", KEYWORD, resourceName),
		}
	}

	return resp
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// 解析 AdmissionReview 请求
	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, "Failed to unmarshal request", http.StatusBadRequest)
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
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func main() {
	// 注册 webhook 处理器
	http.HandleFunc("/validate", handleValidate)

	// 启动服务器
	log.Println("Starting webhook server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
