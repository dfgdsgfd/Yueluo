package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

func (h NativeHandlers) adminNotificationTemplateTest(c *gin.Context) {
	path := c.Request.URL.Path
	id := matrixParam(c, "id")
	var row map[string]any
	if err := h.DB.WithContext(c.Request.Context()).Table("notification_templates").Where("id = ?", id).Take(&row).Error; writeDBError(c, err, "通知模板不存在") {
		return
	}
	vars := notificationSampleVars(h.Config.Notify.Discord.SiteName, h.Config.Notify.Discord.SiteURL)
	if strings.HasSuffix(path, "/test-email") {
		body := readBodyMap(c)
		to := toString(body["email"])
		if to == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供测试邮箱地址", nil)
			return
		}
		if !h.Config.Email.Enabled {
			response.JSON(c, http.StatusBadRequest, response.CodeError, "邮件服务未启用", nil)
			return
		}
		subject := renderTemplate(firstNonEmpty(toString(row["email_subject"]), toString(row["subject"]), "测试邮件"), vars)
		html := renderTemplate(firstNonEmpty(toString(row["email_body"]), toString(row["content"]), "<p>测试邮件内容</p>"), vars)
		if err := h.sendSMTPMail(to, "[测试] "+subject, html); err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "发送失败: "+err.Error(), nil)
			return
		}
		writeSimpleSuccess(c, "测试邮件已发送至 "+to)
		return
	}
	if !h.Config.Notify.Discord.Enabled || h.Config.Notify.Discord.WebhookURL == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeError, "Discord Webhook未启用", nil)
		return
	}
	text := renderTemplate(firstNonEmpty(toString(row["system_template"]), toString(row["content"]), "测试通知"), vars)
	embed := gin.H{
		"title":       "[测试] " + h.Config.Notify.Discord.SiteName,
		"description": text,
		"color":       16776960,
		"footer":      gin.H{"text": "模板: " + firstNonEmpty(toString(row["name"]), id)},
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	if err := h.sendDiscordWebhook(embed); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "发送失败: "+err.Error(), nil)
		return
	}
	writeSimpleSuccess(c, "Discord测试通知已发送")
}

func (h NativeHandlers) sendSMTPMail(to string, subject string, html string) error {
	cfg := h.Config.Email
	from := cfg.FromMail
	if from == "" {
		from = cfg.SMTP.Username
	}
	if from == "" || cfg.SMTP.Host == "" || cfg.SMTP.Username == "" || cfg.SMTP.Password == "" {
		return errors.New("SMTP配置不完整")
	}
	addr := cfg.SMTP.Host + ":" + strconv.Itoa(cfg.SMTP.Port)
	auth := smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Host)
	headers := []string{
		"From: " + cfg.FromName + " <" + from + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="UTF-8"`,
	}
	message := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + html)
	if !cfg.SMTP.Secure {
		return smtp.SendMail(addr, auth, from, []string{to}, message)
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.SMTP.Host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, cfg.SMTP.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(message); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (h NativeHandlers) sendDiscordWebhook(embed gin.H) error {
	body, _ := json.Marshal(gin.H{"embeds": []gin.H{embed}})
	req, err := http.NewRequest(http.MethodPost, h.Config.Notify.Discord.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("discord webhook status %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (h NativeHandlers) adminUserTransferEarnings(c *gin.Context) {
	userID := adminUserIDFromPath(c)
	body := readBodyMap(c)
	amount, ok := float64FromAny(body["amount"])
	if !ok || amount <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "转移金额必须大于0", nil)
		return
	}
	var result gin.H
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var points domain.UserPoints
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&points).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户月币余额不足")
		}
		if err != nil {
			return err
		}
		if points.Points < amount {
			return errors.New("用户月币余额不足")
		}
		newPoints := points.Points - amount
		if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", userID).Update("points", newPoints).Error; err != nil {
			return err
		}
		reason := "管理员将月币转移至创作者收益: " + strconv.FormatFloat(amount, 'f', -1, 64)
		if err := tx.Create(&domain.PointsLog{UserID: userID, Amount: -amount, BalanceAfter: newPoints, Type: repositories.PointsLogTypeTransferToEarnings, Reason: &reason}).Error; err != nil {
			return err
		}
		var earnings domain.CreatorEarnings
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&earnings).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			earnings = domain.CreatorEarnings{UserID: userID}
			if err := tx.Create(&earnings).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		newBalance := earnings.Balance + amount
		newTotal := earnings.TotalEarnings + amount
		if err := tx.Model(&domain.CreatorEarnings{}).Where("user_id = ?", userID).Updates(map[string]any{"balance": newBalance, "total_earnings": newTotal}).Error; err != nil {
			return err
		}
		earnReason := "管理员从月币余额转移至创作者收益: " + strconv.FormatFloat(amount, 'f', -1, 64)
		if err := tx.Create(&domain.CreatorEarningsLog{UserID: userID, EarningsID: earnings.ID, Amount: amount, BalanceAfter: newBalance, Type: "transfer_from_wallet", Reason: &earnReason}).Error; err != nil {
			return err
		}
		result = gin.H{"amount": amount, "newPoints": newPoints, "newEarningsBalance": newBalance}
		return nil
	})
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, "转移成功", result)
}
