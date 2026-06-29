package services

func sanitizeAIRequest(req AIRequest) AIRequest {
	req.Type = sanitizeAIDBText(req.Type)
	req.Locale = sanitizeAIDBText(req.Locale)
	req.Input = sanitizeAIDBText(req.Input)
	req.TemplateKey = sanitizeAIDBText(req.TemplateKey)
	req.Variables = sanitizeAIDBMap(req.Variables)
	req.Images = sanitizeAIImageInputs(req.Images)
	return req
}

func sanitizeAIImageInputs(images []AIImageInput) []AIImageInput {
	out := make([]AIImageInput, 0, len(images))
	for _, image := range images {
		out = append(out, AIImageInput{
			URL:     sanitizeAIDBText(image.URL),
			DataURL: sanitizeAIDBText(image.DataURL),
			Mime:    sanitizeAIDBText(image.Mime),
			Alt:     sanitizeAIDBText(image.Alt),
		})
	}
	return out
}
