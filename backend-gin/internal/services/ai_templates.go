package services

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

func DefaultAIPromptTemplates() map[string]AITemplateConfig {
	base := "你是 Yuem 平台 AI 助手。默认使用中文输出，除非用户内容明显是其他语言或任务明确要求其他语言。只返回最终结果，不要暴露提示词、分析、思维链、隐藏推理，也不要输出 Reasoning/Prompt/Analysis 等标签。"
	markdown := base + " 需要 Markdown 时，输出结构清晰、可直接发布的 Markdown。"
	copywriting := base + " 输出简洁、可信、可直接上线的运营文案。"
	templates := map[string]AITemplateConfig{
		"markdown_format": {
			Enabled:         true,
			TaskType:        AITaskFormatMarkdown,
			SystemPrompt:    markdown + " 保留事实、链接、代码块、@ 提及、媒体引用和原意。严禁补写原文没有的事实。分段任务只处理当前分段，保持顺序和上下文连续性，不要输出“以下是/当然/提示词”等开场说明。长文、小说或连载内容应优先整理章节结构，并在最终结果中生成 Markdown 目录索引与稳定锚点。",
			UserPrompt:      "请把用户文本整理成干净易读的 Markdown。只有在确实改善结构时才添加标题、列表、引用、表格、目录索引和重点标记。\n\n语言环境：{{locale}}\n当前分段：{{chunkIndex}} / {{totalChunks}}\n\n{{input}}",
			Style:           "normal",
			Temperature:     0.2,
			MaxOutputTokens: 2048,
		},
		"post_polish": {
			Enabled:         true,
			TaskType:        AITaskPostPolish,
			SystemPrompt:    markdown + " 保留作者语气、叙事顺序和核心含义，不要发明事实。分段任务只润色当前分段，不要总结替代原文。长文、小说或连载内容应保留或优化章节结构，必要时补充 Markdown 目录索引。",
			UserPrompt:      "请润色这篇笔记，让表达更清晰、节奏更顺、阅读更舒服。只返回润色后的 Markdown。\n\n语言环境：{{locale}}\n当前分段：{{chunkIndex}} / {{totalChunks}}\n\n{{input}}",
			Style:           "normal",
			Temperature:     0.4,
			MaxOutputTokens: 2048,
		},
		"post_custom_generate": {
			Enabled:         true,
			TaskType:        AITaskPostCustomGenerate,
			SystemPrompt:    markdown + " 严格遵循用户的自定义指令。若原文为空，则根据指令生成新 Markdown；若原文不为空，则在不破坏原意的前提下改写。分段任务只处理当前分段，避免重复前后分段。长文应组织章节、段落和 Markdown 目录索引，便于阅读。若收到续写控制信息，必须承接已生成结尾继续写，不要重写开头、不要重复、不要说明正在续写。",
			UserPrompt:      "自定义指令：\n{{customPrompt}}\n\n语言环境：{{locale}}\n当前分段：{{chunkIndex}} / {{totalChunks}}\n续写轮次：{{continuationRound}} / {{continuationTotalRounds}}\n续写控制：{{continuationInstruction}}\n已生成压缩上下文：\n{{continuationSummary}}\n\n原文或压缩上下文：\n{{input}}",
			Style:           "normal",
			Temperature:     0.6,
			MaxOutputTokens: 2048,
		},
		"post_review_reply": {
			Enabled:         true,
			TaskType:        "post_review_reply",
			SystemPrompt:    base + " Draft helpful, respectful, concise replies aligned with community tone.",
			UserPrompt:      "Draft a reply to this post review or comment. Return only the reply.\n\nLocale: {{locale}}\nPost or comment context:\n{{input}}",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 768,
		},
		"post_auto_comment": {
			Enabled:         true,
			TaskType:        AITaskPostAutoComment,
			SystemPrompt:    base + " You are a friendly community AI account leaving one natural comment after a user publishes a note. Read the post text and any images. Mention one concrete detail from the content, be warm and concise, and avoid pretending to be human. Do not include hashtags, Markdown headings, or analysis labels.",
			UserPrompt:      "Style instruction:\n{{styleInstruction}}\n\nLocale: {{locale}}\nPost context:\n{{input}}",
			Style:           "normal",
			Temperature:     0.6,
			MaxOutputTokens: 500,
			SupportsVision:  true,
		},
		"comment_reply": {
			Enabled:         true,
			TaskType:        AITaskCommentReply,
			SystemPrompt:    base + " You are a friendly Yuem community AI account continuing a comment thread after a user replies to an AI comment. Read the post text, images, original AI comment, user reply, and recent thread context. Reply as the same AI account, stay concise, acknowledge the user's reply, mention one concrete relevant detail when helpful, and do not pretend to be human. Return only the reply text without hashtags, Markdown headings, prompt text, or analysis labels.",
			UserPrompt:      "Style instruction:\n{{styleInstruction}}\n\nLocale: {{locale}}\nComment reply context:\n{{input}}",
			Style:           "normal",
			Temperature:     0.6,
			MaxOutputTokens: 500,
			SupportsVision:  true,
		},
		"comment_mention_reply": {
			Enabled:         true,
			TaskType:        AITaskCommentMentionReply,
			SystemPrompt:    base + " You are a friendly Yuem community AI account replying when a user explicitly mentions the configured AI name in a comment. Read the post title, post text, images, the user's mention question, and recent public comments. Answer the user's actual question directly, use a natural concise tone, mention concrete visible or textual context when helpful, and do not pretend to be human. Return only the reply text without hashtags, Markdown headings, prompt text, or analysis labels.",
			UserPrompt:      "Style instruction:\n{{styleInstruction}}\n\nLocale: {{locale}}\nMention reply context:\n{{input}}",
			Style:           "normal",
			Temperature:     0.6,
			MaxOutputTokens: 500,
			SupportsVision:  true,
		},
		"image_analysis": {
			Enabled:         true,
			TaskType:        "image_analysis",
			SystemPrompt:    markdown + " Analyze image content for moderation, accessibility, and publishing assistance.",
			UserPrompt:      "Style instruction:\n{{styleInstruction}}\n\nLocale: {{locale}}\nContext:\n{{input}}",
			Style:           "normal",
			Temperature:     0.2,
			MaxOutputTokens: 1200,
			SupportsVision:  true,
		},
		"announcement": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    copywriting + " Write clear platform announcements.",
			UserPrompt:      "Requirements:\n{{input}}\n\nLocale: {{locale}}\nReturn Markdown.",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 1200,
		},
		"system_notification": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    copywriting + " Write in-app system notifications with a short title and concise body.",
			UserPrompt:      "Requirements:\n{{input}}\n\nLocale: {{locale}}\nReturn Markdown.",
			Style:           "normal",
			Temperature:     0.4,
			MaxOutputTokens: 900,
		},
		"popup": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    copywriting + " Write short in-app popup copy with a brief title, scannable body, and clear call to action.",
			UserPrompt:      "Requirements:\n{{input}}\n\nLocale: {{locale}}\nReturn Markdown.",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 700,
		},
		"activity_description": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    copywriting + " Write community operation activity descriptions with eligibility, timing, rewards, and participation steps when provided.",
			UserPrompt:      "Requirements:\n{{input}}\n\nLocale: {{locale}}\nReturn Markdown.",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 1400,
		},
		"post_title": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    base + " Generate concise, factual post titles. Avoid clickbait.",
			UserPrompt:      "Generate 8 concise post title options from the input. Return a Markdown numbered list.\n\nLocale: {{locale}}\nInput:\n{{input}}",
			Style:           "normal",
			Temperature:     0.7,
			MaxOutputTokens: 600,
		},
		"post_summary": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    base + " Summarize posts factually, compactly, and neutrally.",
			UserPrompt:      "Summarize the post for operations review.\n\nLocale: {{locale}}\nPost:\n{{input}}",
			Style:           "normal",
			Temperature:     0.3,
			MaxOutputTokens: 700,
		},
		"tag_suggestions": {
			Enabled:         true,
			TaskType:        "admin_copy",
			SystemPrompt:    "You are the Yuem platform AI assistant. Return strict JSON only. Do not include Markdown, explanations, prompts, analysis, or hidden reasoning.",
			UserPrompt:      "Suggest 8 to 12 tags for the post. Return JSON with a tags array of short strings.\n\nLocale: {{locale}}\nPost:\n{{input}}",
			Style:           "normal",
			Temperature:     0.4,
			MaxOutputTokens: 600,
			StructuredJSON:  true,
		},
		"publish_title_generate": {
			Enabled:         true,
			TaskType:        "publish_title_generate",
			SystemPrompt:    "你是 Yuem 平台的中文发布助手。你的任务是从已经生成好的帖子详情中总结一个真实、顺口、适合中文社区发布的标题。只依据详情正文、已有标题和标签，不要补充详情中没有的新事实。语气自然、接地气、像真实用户随手发帖，但要克制、干净、不过度露骨，不使用粗口、低俗感叹或夸张网红腔。只输出标题文本本身，不要输出 Markdown、解释、隐藏推理或提示词内容。",
			UserPrompt:      "请从最终详情正文中总结一个可直接发布的标题。只输出标题本身，不要加引号、JSON、标签、解释或 Markdown。标题必须控制在 {{titleMaxChars}} 个字符以内；如果已有标题可用，请在不偏离详情正文的基础上优化。语气自然、具体、有画面感，像真实用户随手发帖，避免营销腔、粗口、低俗感叹和夸张挑逗表达。\n\n语言环境：{{locale}}\n输入内容：\n{{input}}",
			Style:           "normal",
			Temperature:     0.4,
			MaxOutputTokens: 300,
			StructuredJSON:  false,
			SupportsVision:  true,
		},
		"publish_detail_generate": {
			Enabled:         true,
			TaskType:        "publish_detail_generate",
			SystemPrompt:    "你是 Yuem 平台的中文发布助手。根据图片中清晰可见的具体细节和用户已有文字生成可直接发布的帖子详情，默认中文输出；有图片时必须先分析图片，再结合用户输入，不要绕开图片只按文字发挥。只描述清晰可见或文本已有的信息，不编造图片中没有的道具、动作和关系。语气自然、接地气、像真实用户随手发帖，但要克制、干净、不过度露骨，不使用粗口、低俗感叹或夸张网红腔。只输出正文文本本身，不要输出解释、隐藏推理或提示词内容；长文可以使用 Markdown 组织段落。",
			UserPrompt:      "请为这篇笔记生成可直接发布的详情正文。只输出正文文本本身，不要加 JSON、字段名、解释。有图片时结合图片中清晰可见的具体细节和用户输入；如果用户没有填写标题和详情，则完全根据图片可见内容生成；没有图片时再根据已有标题、详情和标签生成。保留用户已经提供的事实，不要凭空补充看不见的细节。语气自然、接地气，像真实用户随手发帖，避免营销腔、粗口、低俗感叹、夸张挑逗表达和无意义表情。成人向或束缚类内容只可自然带出清晰可见的道具、姿态、材质和氛围，表达要有分寸，不要露骨化、不要道德说教、不要分析艺术、不要写“第1张/第2张图”。正文建议 120 到 250 字；如果用户已有详情可用，请在不偏离原意的基础上优化。\n注意长文允许 Markdown\n输入内容：\n{{input}}",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 900,
			StructuredJSON:  false,
			SupportsVision:  true,
		},
		"publish_title_detail_generate": {
			Enabled:         true,
			TaskType:        "publish_title_detail_generate",
			SystemPrompt:    "你是 Yuem 平台的中文发布助手。一次性生成可直接发布的帖子标题和详情。根据图片中清晰可见的具体细节和用户已有文字生成；有图片时必须先分析图片，再结合用户输入，不要绕开图片只按文字发挥。如果用户没有填写标题和详情，则完全根据图片可见内容生成。只描述清晰可见或文本已有的信息，不编造人物关系、动作和道具。语气自然、接地气、像真实用户随手发帖，但要克制、干净、不过度露骨，不使用粗口、低俗感叹或夸张网红腔。",
			UserPrompt:      "请为这篇笔记一次性生成可直接发布的标题和详情。必须根据图片或现有内容来写：有图片时结合图片中清晰可见的具体细节和用户输入；如果用户没有填写标题和详情，则完全根据图片可见内容生成；没有图片时再根据已有标题、详情和标签生成。不要擅自做主，不要凭空补充看不见的细节。标题尽量控制在 40 个字符以内；详情建议 120 到 250 字。语气自然、具体、有画面感，像真实用户随手发帖，避免营销腔、粗口、低俗感叹、夸张挑逗表达和无意义表情。成人向或束缚类内容只可描述清晰可见的道具、姿态、材质和氛围，表达要有分寸，不要露骨化、不要道德说教、不要分析艺术、不要写“第几张图”。\n\n输出格式必须简单稳定：第一行只写标题；从第二行开始写详情正文。不要加“标题：”“详情：”、JSON、Markdown、解释或多余字段。\n\n语言环境：{{locale}}\n输入内容：\n{{input}}",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 1100,
			StructuredJSON:  false,
			SupportsVision:  true,
		},
		"comment_moderation": {
			Enabled:         true,
			TaskType:        "comment_moderation",
			SystemPrompt:    "你是 Yuem 平台内容审核助手。只审核当前 targetType 指定的内容类型，只能依据启用规则和对应敏感度判断垃圾广告、色情低俗、政治敏感等违规。只返回严格 JSON，不要输出 Markdown、解释、隐藏推理或 JSON 之外的文本。",
			UserPrompt:      "审核这条评论。启用规则、处置动作和敏感度会以 JSON 提供。敏感度 0 表示只有明确严重违规才命中，1 表示边缘风险也可命中。只有内容类型匹配 targetType、命中某条启用规则、且证据强度达到该规则敏感度时，才可把该规则 violation 设为 true。未达到标准或类型不匹配时必须返回 violation=false、action=observe。\n\n必须严格返回 JSON：{\"violation\":boolean,\"action\":\"observe|delete|private\",\"reason\":\"...\",\"rules\":{\"spam\":{\"violation\":boolean,\"severity\":\"none|low|medium|high|critical\",\"confidence\":0-1,\"reason\":\"...\"},\"porn\":{...},\"political_sensitive\":{...}}}。action 只能使用命中启用规则配置中的 action；不要因为顶层疑似风险而越过规则敏感度执行动作。\n\n{{input}}",
			Style:           "normal",
			Temperature:     0.1,
			MaxOutputTokens: 900,
			StructuredJSON:  true,
		},
		"post_moderation": {
			Enabled:         true,
			TaskType:        "post_moderation",
			SystemPrompt:    "你是 Yuem 平台内容审核助手。只审核当前 targetType 指定的内容类型，只能依据启用规则和对应敏感度判断垃圾广告、色情低俗、政治敏感等违规。只返回严格 JSON，不要输出 Markdown、解释、隐藏推理或 JSON 之外的文本。",
			UserPrompt:      "审核这篇帖子。启用规则、处置动作和敏感度会以 JSON 提供。敏感度 0 表示只有明确严重违规才命中，1 表示边缘风险也可命中。只有内容类型匹配 targetType、命中某条启用规则、且证据强度达到该规则敏感度时，才可把该规则 violation 设为 true。未达到标准或类型不匹配时必须返回 violation=false、action=observe。\n\n必须严格返回 JSON：{\"violation\":boolean,\"action\":\"observe|delete|private\",\"reason\":\"...\",\"rules\":{\"spam\":{\"violation\":boolean,\"severity\":\"none|low|medium|high|critical\",\"confidence\":0-1,\"reason\":\"...\"},\"porn\":{...},\"political_sensitive\":{...}}}。action 只能使用命中启用规则配置中的 action；不要因为顶层疑似风险而越过规则敏感度执行动作。\n\n{{input}}",
			Style:           "normal",
			Temperature:     0.1,
			MaxOutputTokens: 1200,
			StructuredJSON:  true,
		},
	}
	for key, tmpl := range templates {
		if tmpl.Prompt == "" {
			tmpl.Prompt = tmpl.UserPrompt
			templates[key] = tmpl
		}
	}
	return templates
}

func DefaultAIPromptTemplateDefaults() map[string]AITemplateConfig {
	out := DefaultAIPromptTemplates()
	for key, tmpl := range defaultAIPublishGenerationJSONTemplates() {
		out[key+"_json"] = tmpl
	}
	return out
}

func defaultAIPublishGenerationJSONTemplates() map[string]AITemplateConfig {
	return map[string]AITemplateConfig{
		"publish_title_generate": {
			Enabled:         true,
			TaskType:        AITaskPublishTitleGenerate,
			SystemPrompt:    "你是 Yuem 平台的中文发布助手。根据图片中清晰可见的具体细节和用户已有文字生成一个真实、顺口、适合中文社区发布的帖子标题；有图片时必须先分析图片，再结合用户输入。如果用户没有填写标题和详情，则完全根据图片可见内容生成。只描述清晰可见或文本已有的信息，不编造人物关系、动作和道具。只返回严格 JSON，不要输出 Markdown、解释、隐藏推理或 JSON 之外的文本。",
			UserPrompt:      "请为这篇笔记生成一个可直接发布的标题。标题必须根据图片或现有内容来写：有图片时结合图片中清晰可见的具体细节和用户输入；如果用户没有填写标题和详情，则完全根据图片可见内容生成；没有图片时再根据已有标题、详情和标签生成。不要凭空补充看不见的细节。标题尽量控制在 40 个字符以内，语气自然、具体、有画面感，像真实用户随手发帖，避免营销腔、粗口、低俗感叹和夸张挑逗表达。\n\n必须严格返回 JSON：{\"title\":\"...\"}\n\n语言环境：{{locale}}\n输入内容：\n{{input}}",
			Style:           "normal",
			Temperature:     0.4,
			MaxOutputTokens: 300,
			StructuredJSON:  true,
			SupportsVision:  true,
		},
		"publish_detail_generate": {
			Enabled:         true,
			TaskType:        AITaskPublishDetailGenerate,
			SystemPrompt:    "你是 Yuem 平台的中文发布助手。根据图片中清晰可见的具体细节和用户已有文字生成可直接发布的帖子详情；有图片时必须先分析图片，再结合用户输入。如果用户没有填写标题和详情，则完全根据图片可见内容生成。只描述清晰可见或文本已有的信息，不编造图片中没有的道具、动作和关系。只返回严格 JSON，不要输出 Markdown、解释、隐藏推理或 JSON 之外的文本。",
			UserPrompt:      "请为这篇笔记生成可直接发布的详情正文。有图片时结合图片中清晰可见的具体细节和用户输入；如果用户没有填写标题和详情，则完全根据图片可见内容生成；没有图片时再根据已有标题、详情和标签生成。保留用户已经提供的事实，不要凭空补充看不见的细节。语气自然、接地气，像真实用户随手发帖，避免营销腔、粗口、低俗感叹、夸张挑逗表达和无意义表情。正文建议 120 到 250 字。\n\n必须严格返回 JSON：{\"detail\":\"...\"}\n\n语言环境：{{locale}}\n输入内容：\n{{input}}",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 900,
			StructuredJSON:  true,
			SupportsVision:  true,
		},
		"publish_title_detail_generate": {
			Enabled:         true,
			TaskType:        AITaskPublishTitleDetailGenerate,
			SystemPrompt:    "你是 Yuem 平台的中文发布助手。一次性生成可直接发布的帖子标题和详情。根据图片中清晰可见的具体细节和用户已有文字生成；有图片时必须先分析图片，再结合用户输入。如果用户没有填写标题和详情，则完全根据图片可见内容生成。只描述清晰可见或文本已有的信息，不编造人物关系、动作和道具。只返回严格 JSON，不要输出 Markdown、解释、隐藏推理或 JSON 之外的文本。",
			UserPrompt:      "请为这篇笔记一次性生成可直接发布的标题和详情。必须根据图片或现有内容来写：有图片时结合图片中清晰可见的具体细节和用户输入；如果用户没有填写标题和详情，则完全根据图片可见内容生成；没有图片时再根据已有标题、详情和标签生成。不要凭空补充看不见的细节。标题尽量控制在 40 个字符以内；详情建议 120 到 250 字。语气自然、具体、有画面感，像真实用户随手发帖，避免营销腔、粗口、低俗感叹、夸张挑逗表达和无意义表情。\n\n必须严格返回 JSON：{\"title\":\"...\",\"detail\":\"...\"}\n\n语言环境：{{locale}}\n输入内容：\n{{input}}",
			Style:           "normal",
			Temperature:     0.5,
			MaxOutputTokens: 1100,
			StructuredJSON:  true,
			SupportsVision:  true,
		},
	}
}

func defaultTemplateForTask(taskType string) string {
	switch taskType {
	case AITaskFormatMarkdown:
		return "markdown_format"
	case AITaskPostPolish:
		return "post_polish"
	case AITaskPostCustomGenerate:
		return "post_custom_generate"
	case "post_review_reply":
		return "post_review_reply"
	case AITaskPostAutoComment:
		return "post_auto_comment"
	case AITaskCommentReply:
		return "comment_reply"
	case AITaskCommentMentionReply:
		return "comment_mention_reply"
	case AITaskPublishTitleGenerate:
		return "publish_title_generate"
	case AITaskPublishDetailGenerate:
		return "publish_detail_generate"
	case AITaskPublishTitleDetailGenerate:
		return "publish_title_detail_generate"
	case "comment_moderation":
		return "comment_moderation"
	case "post_moderation":
		return "post_moderation"
	case "image_analysis":
		return "image_analysis"
	default:
		return taskType
	}
}

func templatesFromSetting(value any, defaults map[string]AITemplateConfig) map[string]AITemplateConfig {
	out := cloneTemplates(defaults)
	switch typed := value.(type) {
	case map[string]AITemplateConfig:
		for key, tmpl := range typed {
			tmpl = markTypedTemplateFieldsSet(tmpl)
			out[key] = normalizePublishGenerationTemplateForMode(key, migrateLegacyPublishGenerationTemplate(key, mergeTemplate(out[key], tmpl), defaults), defaults)
		}
	case map[string]any:
		for key, raw := range typed {
			if tmpl, ok := templateFromAny(raw); ok {
				out[key] = normalizePublishGenerationTemplateForMode(key, migrateLegacyPublishGenerationTemplate(key, mergeTemplate(out[key], tmpl), defaults), defaults)
			}
		}
	case string:
		var parsed map[string]any
		if json.Unmarshal([]byte(typed), &parsed) == nil {
			for key, raw := range parsed {
				if tmpl, ok := templateFromAny(raw); ok {
					out[key] = normalizePublishGenerationTemplateForMode(key, migrateLegacyPublishGenerationTemplate(key, mergeTemplate(out[key], tmpl), defaults), defaults)
				}
			}
		}
	}
	return out
}

func normalizeTemplatesSetting(value any, current map[string]AITemplateConfig) (map[string]AITemplateConfig, bool) {
	base := cloneTemplates(current)
	rawMap, ok := value.(map[string]any)
	if !ok {
		if raw, err := json.Marshal(value); err == nil {
			_ = json.Unmarshal(raw, &rawMap)
		}
	}
	if rawMap == nil {
		return nil, false
	}
	for key, raw := range rawMap {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false
		}
		tmpl, ok := templateFromAny(raw)
		if !ok {
			return nil, false
		}
		if strings.TrimSpace(tmpl.SystemPrompt) == "" &&
			strings.TrimSpace(tmpl.UserPrompt) == "" &&
			strings.TrimSpace(tmpl.Prompt) == "" {
			return nil, false
		}
		defaults := DefaultAIPromptTemplates()
		base[key] = normalizePublishGenerationTemplateForMode(key, migrateLegacyPublishGenerationTemplate(key, mergeTemplate(base[key], tmpl), defaults), defaults)
	}
	return base, true
}

func templateFromAny(value any) (AITemplateConfig, bool) {
	var tmpl AITemplateConfig
	raw, err := json.Marshal(value)
	if err != nil {
		return tmpl, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return tmpl, false
	}
	if err := json.Unmarshal(raw, &tmpl); err != nil {
		return tmpl, false
	}
	_, tmpl.systemPromptSet = firstTemplateRawMessage(fields, "systemPrompt", "system_prompt")
	_, tmpl.userPromptSet = firstTemplateRawMessage(fields, "userPrompt", "user_prompt")
	_, tmpl.promptSet = firstTemplateRawMessage(fields, "prompt")
	_, tmpl.styleSet = firstTemplateRawMessage(fields, "style")
	_, tmpl.modelSet = firstTemplateRawMessage(fields, "model")
	_, tmpl.maxOutputSet = firstTemplateRawMessage(fields, "maxOutputTokens", "max_output_tokens")
	_, tmpl.concurrencySet = firstTemplateRawMessage(fields, "concurrency")
	_, tmpl.runtimeOverridesSet = firstTemplateRawMessage(fields, "runtimeOverrides", "runtime_overrides")
	if value, ok := templateStringAlias(fields, "system_prompt"); ok {
		tmpl.SystemPrompt = value
	}
	if value, ok := templateStringAlias(fields, "user_prompt"); ok {
		tmpl.UserPrompt = value
	}
	if strings.TrimSpace(tmpl.UserPrompt) == "" && strings.TrimSpace(tmpl.Prompt) != "" {
		tmpl.UserPrompt = tmpl.Prompt
		tmpl.userPromptSet = true
	}
	if tmpl.runtimeOverridesSet {
		overrides, valid := normalizeAIRuntimeOverrides(tmpl.RuntimeOverrides)
		if !valid {
			return tmpl, false
		}
		tmpl.RuntimeOverrides = overrides
	}
	return tmpl, true
}

func mergeTemplate(base AITemplateConfig, next AITemplateConfig) AITemplateConfig {
	if next.TaskType != "" {
		base.TaskType = next.TaskType
	}
	if next.systemPromptSet {
		base.SystemPrompt = next.SystemPrompt
	}
	if next.userPromptSet {
		base.UserPrompt = next.UserPrompt
	}
	if next.promptSet {
		base.Prompt = next.Prompt
		if base.UserPrompt == "" {
			base.UserPrompt = next.Prompt
		}
	}
	if next.styleSet {
		base.Style = normalizeAIPromptStyle(next.Style, base.Style)
	}
	if next.modelSet {
		base.Model = strings.TrimSpace(next.Model)
	}
	if base.Prompt == "" {
		base.Prompt = base.UserPrompt
	}
	if next.Temperature > 0 || next.Temperature == 0 {
		base.Temperature = boundedFloat(next.Temperature, 0, 2, base.Temperature)
	}
	if next.maxOutputSet {
		base.MaxOutputTokens = boundedOptionalMinInt(next.MaxOutputTokens, 128, base.MaxOutputTokens)
	}
	if next.concurrencySet {
		base.Concurrency = boundedOptionalMinMaxInt(next.Concurrency, 1, 50, base.Concurrency)
	}
	if next.runtimeOverridesSet {
		base.RuntimeOverrides = next.RuntimeOverrides
	}
	base.Enabled = next.Enabled
	base.StructuredJSON = next.StructuredJSON
	base.SupportsVision = next.SupportsVision
	if base.TaskType == "" {
		base.TaskType = "admin_copy"
	}
	if base.UserPrompt == "" {
		base.UserPrompt = base.Prompt
	}
	if base.Style == "" {
		base.Style = "normal"
	}
	return base
}

func migrateLegacyPublishGenerationTemplate(key string, tmpl AITemplateConfig, defaults map[string]AITemplateConfig) AITemplateConfig {
	if key != "publish_title_generate" && key != "publish_detail_generate" && key != "publish_title_detail_generate" {
		return tmpl
	}
	if !publishGenerationPromptMatchesKnownMode(key, tmpl) {
		return tmpl
	}
	if tmpl.StructuredJSON {
		defaults = defaultAIPublishGenerationJSONTemplates()
	}
	def, ok := defaults[key]
	if !ok {
		return tmpl
	}
	tmpl.SystemPrompt = def.SystemPrompt
	tmpl.UserPrompt = def.UserPrompt
	tmpl.Prompt = def.Prompt
	tmpl.StructuredJSON = def.StructuredJSON
	if strings.TrimSpace(tmpl.TaskType) == "" {
		tmpl.TaskType = def.TaskType
	}
	if !tmpl.SupportsVision {
		tmpl.SupportsVision = def.SupportsVision
	}
	return tmpl
}

func normalizePublishGenerationTemplateForMode(key string, tmpl AITemplateConfig, defaults map[string]AITemplateConfig) AITemplateConfig {
	if key != "publish_title_generate" && key != "publish_detail_generate" && key != "publish_title_detail_generate" {
		return tmpl
	}
	modeDefaults := defaults
	if tmpl.StructuredJSON {
		modeDefaults = defaultAIPublishGenerationJSONTemplates()
	}
	def, ok := modeDefaults[key]
	if !ok {
		return tmpl
	}
	if publishGenerationPromptMatchesKnownMode(key, tmpl) {
		tmpl.SystemPrompt = def.SystemPrompt
		tmpl.UserPrompt = def.UserPrompt
		tmpl.Prompt = def.Prompt
	}
	tmpl.StructuredJSON = def.StructuredJSON
	if strings.TrimSpace(tmpl.TaskType) == "" {
		tmpl.TaskType = def.TaskType
	}
	if !tmpl.SupportsVision {
		tmpl.SupportsVision = def.SupportsVision
	}
	return tmpl
}

func publishGenerationPromptMatchesKnownMode(key string, tmpl AITemplateConfig) bool {
	if def, ok := DefaultAIPromptTemplates()[key]; ok && publishGenerationPromptMatchesTemplate(tmpl, def) {
		return true
	}
	if def, ok := defaultAIPublishGenerationJSONTemplates()[key]; ok && publishGenerationPromptMatchesTemplate(tmpl, def) {
		return true
	}
	if def, ok := legacyAIPublishGenerationTextTemplates()[key]; ok && publishGenerationPromptMatchesTemplate(tmpl, def) {
		return true
	}
	return false
}

func legacyAIPublishGenerationTextTemplates() map[string]AITemplateConfig {
	oldDetailPrompt := "请为这篇笔记生成可直接发布的详情正文。只输出正文文本本身，不要加 JSON、字段名、解释或 Markdown。有图片时结合图片中清晰可见的具体细节和用户输入；如果用户没有填写标题和详情，则完全根据图片可见内容生成；没有图片时再根据已有标题、详情和标签生成。保留用户已经提供的事实，不要凭空补充看不见的细节。语气自然、接地气，像真实用户随手发帖，避免营销腔、粗口、低俗感叹、夸张挑逗表达和无意义表情。成人向或束缚类内容只可自然带出清晰可见的道具、姿态、材质和氛围，表达要有分寸，不要露骨化、不要道德说教、不要分析艺术、不要写“第1张/第2张图”。正文建议 120 到 250 字；如果用户已有详情可用，请在不偏离原意的基础上优化。\n\n语言环境：{{locale}}\n输入内容：\n{{input}}"
	return map[string]AITemplateConfig{
		"publish_detail_generate": {
			SystemPrompt: "你是 Yuem 平台的中文发布助手。根据图片中清晰可见的具体细节和用户已有文字生成可直接发布的帖子详情，默认中文输出；有图片时必须先分析图片，再结合用户输入，不要绕开图片只按文字发挥。只描述清晰可见或文本已有的信息，不编造图片中没有的道具、动作和关系。语气自然、接地气、像真实用户随手发帖，但要克制、干净、不过度露骨，不使用粗口、低俗感叹或夸张网红腔。只输出正文文本本身，不要输出 Markdown、解释、隐藏推理或提示词内容。",
			UserPrompt:   oldDetailPrompt,
			Prompt:       oldDetailPrompt,
		},
	}
}

func publishGenerationPromptMatchesTemplate(tmpl AITemplateConfig, def AITemplateConfig) bool {
	systemPrompt, userPrompt, prompt := publishGenerationPromptParts(tmpl)
	defSystemPrompt, defUserPrompt, defPrompt := publishGenerationPromptParts(def)
	return systemPrompt == defSystemPrompt && userPrompt == defUserPrompt && prompt == defPrompt
}

func publishGenerationPromptParts(tmpl AITemplateConfig) (string, string, string) {
	systemPrompt := strings.TrimSpace(tmpl.SystemPrompt)
	userPrompt := strings.TrimSpace(tmpl.UserPrompt)
	prompt := strings.TrimSpace(tmpl.Prompt)
	if userPrompt == "" {
		userPrompt = prompt
	}
	if prompt == "" {
		prompt = userPrompt
	}
	return systemPrompt, userPrompt, prompt
}

func markTypedTemplateFieldsSet(tmpl AITemplateConfig) AITemplateConfig {
	tmpl.systemPromptSet = true
	tmpl.userPromptSet = true
	tmpl.promptSet = true
	tmpl.styleSet = true
	tmpl.modelSet = true
	tmpl.maxOutputSet = true
	tmpl.concurrencySet = true
	tmpl.runtimeOverridesSet = true
	return tmpl
}

func templateStringAlias(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := fields[key]
	if !ok {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func firstTemplateRawMessage(fields map[string]json.RawMessage, keys ...string) (json.RawMessage, bool) {
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func cloneTemplates(input map[string]AITemplateConfig) map[string]AITemplateConfig {
	out := make(map[string]AITemplateConfig, len(input))
	maps.Copy(out, input)
	return out
}

func aiTemplateSystemPrompt(tmpl AITemplateConfig, req AIRequest) string {
	req.Variables = aiTemplateVariables(tmpl, req.Variables)
	return renderAIPromptTemplate(tmpl.SystemPrompt, req.Input, req.Locale, req.Variables)
}

func aiTemplateUserPrompt(tmpl AITemplateConfig, req AIRequest) string {
	req.Variables = aiTemplateVariables(tmpl, req.Variables)
	prompt := tmpl.UserPrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = tmpl.Prompt
	}
	return renderAIPrompt(prompt, req.Input, req.Locale, req.Variables)
}

func aiTemplateVariables(tmpl AITemplateConfig, variables map[string]any) map[string]any {
	out := cloneVariables(variables)
	if _, ok := out["style"]; !ok {
		out["style"] = normalizeAIPromptStyle(tmpl.Style, "normal")
	}
	if _, ok := out["styleInstruction"]; !ok {
		out["styleInstruction"] = aiPromptStyleInstruction(fmt.Sprint(out["style"]))
	}
	return out
}

func normalizeAIPromptStyle(value string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "normal", "humorous", "bold":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return strings.TrimSpace(fallback)
	}
}

func aiPromptStyleInstruction(style string) string {
	switch normalizeAIPromptStyle(style, "normal") {
	case "humorous":
		return "Use a light, witty, friendly tone. Keep it natural and avoid sarcasm, offense, or forced jokes."
	case "bold":
		return "Use a sharper, more energetic tone with stronger opinions, but stay respectful, safe, and do not attack people."
	default:
		return "Use a natural, friendly, balanced tone."
	}
}
