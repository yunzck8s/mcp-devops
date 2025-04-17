package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings" // Import strings package
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// WeChat 相关常量
const (
	weChatTimeout       = 10 * time.Second
	defaultTemplateCard = "text_notice"
)

// WeChatMessage 企业微信消息结构
type WeChatMessage struct {
	MsgType      string                 `json:"msgtype"`
	AgentID      string                 `json:"agentid,omitempty"`
	TemplateCard *WeChatTemplateCard    `json:"template_card,omitempty"`
	Text         *WeChatTextMessage     `json:"text,omitempty"`
	Markdown     *WeChatMarkdownMessage `json:"markdown,omitempty"`
}

// WeChatTemplateCard 模板卡片消息
type WeChatTemplateCard struct {
	CardType              string                    `json:"card_type"`
	Source                *WeChatTemplateCardSource `json:"source,omitempty"`
	MainTitle             *WeChatTemplateCardTitle  `json:"main_title"`
	HorizontalContentList []HorizontalContent       `json:"horizontal_content_list,omitempty"`
}

// WeChatTemplateCardSource 模板卡片来源信息
type WeChatTemplateCardSource struct {
	IconURL string `json:"icon_url,omitempty"`
	Desc    string `json:"desc,omitempty"`
}

// WeChatTemplateCardTitle 模板卡片标题
type WeChatTemplateCardTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

// HorizontalContent 卡片内容条目
type HorizontalContent struct {
	KeyName string `json:"keyname"`
	Value   string `json:"value"`
}

// WeChatTextMessage 文本消息
type WeChatTextMessage struct {
	Content string `json:"content"`
}

// WeChatMarkdownMessage Markdown消息
type WeChatMarkdownMessage struct {
	Content string `json:"content"`
}

// SendWeChatMessageTool 发送企业微信消息工具
func SendWeChatMessageTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fmt.Println("ai 正在调用mcp server的tool: send_wechat_message")

	// 获取并验证参数
	msgType, ok := request.Params.Arguments["msg_type"].(string)
	if !ok {
		msgType = "text" // 默认为文本消息
	}

	content, ok := request.Params.Arguments["content"].(string)
	if !ok || content == "" {
		return mcp.NewToolResultText("消息内容不能为空"), fmt.Errorf("消息内容不能为空")
	}

	// 获取标题，为 text 类型也设置默认标题
	title, _ := request.Params.Arguments["title"].(string)
	if title == "" {
		title = "系统通知" // 为 text 和 markdown 设置默认标题
		if msgType == "template_card" {
			title = "K8s集群通知" // template_card 使用不同的默认标题
		}
	}

	webhookURL := os.Getenv("WECHAT_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL, _ = request.Params.Arguments["webhook_url"].(string)
		if webhookURL == "" {
			return mcp.NewToolResultText("企业微信Webhook URL未设置，请配置WECHAT_WEBHOOK_URL环境变量或在参数中提供webhook_url"),
				fmt.Errorf("企业微信Webhook URL未设置，请配置WECHAT_WEBHOOK_URL环境变量或在参数中提供webhook_url")
		}
	}

	// 构建消息
	var message WeChatMessage

	switch msgType {
	case "text":
		// 当类型为 text 时，自动转换为 Markdown 格式以支持颜色
		var color = "info" // 默认颜色为 info (蓝色)
		lowerContent := strings.ToLower(content)
		// 基本的关键字判断来决定颜色
		if strings.Contains(lowerContent, "error") || strings.Contains(lowerContent, "fail") || strings.Contains(lowerContent, "故障") || strings.Contains(lowerContent, "失败") || strings.Contains(lowerContent, "critical") || strings.Contains(lowerContent, "firing") {
			color = "warning" // warning (橙色)
		} else if strings.Contains(lowerContent, "success") || strings.Contains(lowerContent, "成功") || strings.Contains(lowerContent, "resolved") || strings.Contains(lowerContent, "已解决") {
			color = "info" // info (蓝色) - 也可以用 comment (灰色) 或其他
		}

		// 构建 Markdown 内容
		markdownContent := fmt.Sprintf("**%s**\n\n<font color=\"%s\">%s</font>\n\n<font color=\"comment\">%s</font>",
			title,
			color,
			content,
			time.Now().Format("2006-01-02 15:04:05"),
		)

		message = WeChatMessage{
			MsgType: "markdown", // 发送类型改为 markdown
			Markdown: &WeChatMarkdownMessage{
				Content: markdownContent,
			},
		}
	case "markdown":
		// 如果用户明确指定 markdown，则直接使用提供的 content
		message = WeChatMessage{
			MsgType: "markdown",
			Markdown: &WeChatMarkdownMessage{
				Content: content,
			},
		}
	case "template_card":
		cardType, _ := request.Params.Arguments["card_type"].(string)
		if cardType == "" {
			cardType = defaultTemplateCard
		}

		// 创建卡片内容
		horizontalContents := []HorizontalContent{
			{KeyName: "内容", Value: content},
		}

		// 如果有时间参数，添加到卡片
		timeStr := time.Now().Format("2006-01-02 15:04:05")

		horizontalContents = append(horizontalContents, HorizontalContent{
			KeyName: "时间",
			Value:   timeStr,
		})

		message = WeChatMessage{
			MsgType: "template_card",
			TemplateCard: &WeChatTemplateCard{
				CardType: cardType,
				Source: &WeChatTemplateCardSource{
					Desc: "Kubernetes管理系统",
				},
				MainTitle: &WeChatTemplateCardTitle{
					Title: title,
				},
				HorizontalContentList: horizontalContents,
			},
		}
	default:
		return mcp.NewToolResultText(fmt.Sprintf("不支持的消息类型: %s", msgType)),
			fmt.Errorf("不支持的消息类型: %s", msgType)
	}

	// 将消息转换为JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("JSON编码失败: %v", err)), err
	}

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: weChatTimeout,
	}

	// 发送请求
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(messageJSON))
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("发送请求失败: %v", err)), err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("读取响应失败: %v", err)), err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultText(fmt.Sprintf("发送失败，状态码: %d，响应: %s", resp.StatusCode, string(body))),
			fmt.Errorf("发送失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("解析响应失败: %v", err)), err
	}

	// 检查企业微信API返回的错误码
	errCode, ok := responseData["errcode"].(float64)
	if !ok || errCode != 0 {
		errMsg, _ := responseData["errmsg"].(string)
		return mcp.NewToolResultText(fmt.Sprintf("企业微信API返回错误: code=%v, msg=%s", errCode, errMsg)),
			fmt.Errorf("企业微信API返回错误: code=%v", errCode)
	}

	// 返回成功结果
	return mcp.NewToolResultText(fmt.Sprintf("企业微信消息发送成功，类型: %s", msgType)), nil
}
