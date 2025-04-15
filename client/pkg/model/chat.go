package model

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// NewChatModel 创建新的聊天模型
func NewChatModel(ctx context.Context, apiKey, baseURL, modelName string) model.ChatModel {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	if modelName == "" {
		modelName = "gpt-4"
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 使用OpenAI提供的模型创建函数
	cm, err := openai.NewChatModel(timeoutCtx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelName,
		BaseURL: baseURL,
	})
	if err != nil {
		log.Fatalf("初始化OpenAI模型失败: %v", err)
	}
	return cm
}
