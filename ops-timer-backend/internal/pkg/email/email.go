package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"mime"
	"net"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"ops-timer-backend/internal/config"
)

// Service 封装 SMTP 邮件发送能力
type Service struct {
	cfg *config.SMTPConfig
}

func NewService(cfg *config.SMTPConfig) *Service {
	return &Service{cfg: cfg}
}

// Enabled 是否已启用邮件通知
func (s *Service) Enabled() bool {
	return s.cfg.Enabled()
}

// NotificationData 是渲染通知邮件所需的数据
type NotificationData struct {
	UnitTitle   string
	UnitType    string
	Message     string
	Level       string
	LevelLabel  string
	LevelEmoji  string
	HeaderColor string
	MsgBgColor  string
	TriggeredAt string
	Details     []Detail
}

type Detail struct {
	Label string
	Value string
}

// SendNotification 发送通知邮件
func (s *Service) SendNotification(to, unitTitle, unitType, message, level string, details []Detail) error {
	data := buildData(unitTitle, unitType, message, level, details)
	html, err := renderNotification(data)
	if err != nil {
		return fmt.Errorf("render email: %w", err)
	}
	subject := fmt.Sprintf("【任务管理器 · %s】%s", data.LevelLabel, unitTitle)
	return s.send(to, subject, html)
}

// SendTest 发送测试邮件
func (s *Service) SendTest(to string) error {
	html := renderTest(to)
	return s.send(to, "【任务管理器】邮件通知测试", html)
}

// ---------- internal ----------

func buildData(unitTitle, unitType, message, level string, details []Detail) NotificationData {
	d := NotificationData{
		UnitTitle:   unitTitle,
		UnitType:    unitType,
		Message:     message,
		Level:       level,
		TriggeredAt: time.Now().Format("2006-01-02 15:04"),
		Details:     details,
	}
	switch level {
	case "critical":
		d.LevelLabel = "紧急通知"
		d.LevelEmoji = "🔴"
		d.HeaderColor = "#D32F2F"
		d.MsgBgColor = "#FFF3F3"
	case "warning":
		d.LevelLabel = "预警通知"
		d.LevelEmoji = "🟡"
		d.HeaderColor = "#E65100"
		d.MsgBgColor = "#FFF8F0"
	default:
		d.LevelLabel = "信息通知"
		d.LevelEmoji = "🔵"
		d.HeaderColor = "#1565C0"
		d.MsgBgColor = "#F0F4FF"
	}
	return d
}

func (s *Service) send(to, subject, htmlBody string) error {
	msg := buildMIME(s.fromAddr(), to, subject, htmlBody)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.Port == 465 {
		return s.sendSSL(addr, to, msg)
	}
	return s.sendSTARTTLS(addr, to, msg)
}

func (s *Service) fromAddr() string {
	re := regexp.MustCompile(`<([^>]+)>`)
	if m := re.FindStringSubmatch(s.cfg.From); len(m) == 2 {
		return m[1]
	}
	return s.cfg.From
}

func (s *Service) sendSTARTTLS(addr, to string, msg []byte) error {
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	return smtp.SendMail(addr, auth, s.fromAddr(), []string{to}, msg)
}

func (s *Service) sendSSL(addr, to string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, host)
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(s.fromAddr()); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

func buildMIME(from, to, subject, htmlBody string) []byte {
	var buf strings.Builder
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	// RFC 2047：对包含非 ASCII 字符的头字段进行 Q 编码，同时防止 CRLF 注入
	buf.WriteString(fmt.Sprintf("From: %s\r\n", sanitizeHeaderValue(from)))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", sanitizeHeaderValue(to)))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", sanitizeHeaderValue(subject))))
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)
	return []byte(buf.String())
}

// sanitizeHeaderValue 清除邮件头中的 CR/LF 字符，防止 SMTP 头注入攻击
func sanitizeHeaderValue(s string) string {
	return strings.NewReplacer("\r", "", "\n", "", "\t", " ").Replace(strings.TrimSpace(s))
}

// ---------- HTML templates ----------

const notificationTmpl = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>任务管理器 通知</title>
</head>
<body style="margin:0;padding:0;background:#F0F2F5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Hiragino Sans GB','Microsoft YaHei',sans-serif">
<table width="100%" cellpadding="0" cellspacing="0" border="0">
  <tr><td align="center" style="padding:40px 16px">

    <table width="600" cellpadding="0" cellspacing="0" border="0" style="max-width:600px;background:#FFFFFF;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,0.10)">

      <!-- ===== HEADER ===== -->
      <tr>
        <td style="background:{{.HeaderColor}};padding:0">
          <table width="100%" cellpadding="0" cellspacing="0" border="0">
            <tr>
              <td style="padding:28px 32px 24px">
                <div style="color:rgba(255,255,255,0.75);font-size:12px;text-transform:uppercase;letter-spacing:2px;font-weight:600;margin-bottom:6px">Task Manager</div>
                <div style="color:#FFFFFF;font-size:26px;font-weight:700;line-height:1.2">{{.LevelEmoji}}&nbsp;{{.LevelLabel}}</div>
              </td>
              <td align="right" style="padding:28px 32px 24px;vertical-align:top">
                <div style="background:rgba(255,255,255,0.18);border-radius:8px;padding:7px 14px;color:#FFFFFF;font-size:12px;white-space:nowrap;font-weight:500">{{.TriggeredAt}}</div>
              </td>
            </tr>
          </table>
          <!-- Wave divider -->
          <div style="height:8px;background:linear-gradient(to bottom,rgba(0,0,0,0.08),transparent)"></div>
        </td>
      </tr>

      <!-- ===== BODY ===== -->
      <tr>
        <td style="padding:32px">

          <!-- Unit title -->
          <div style="font-size:22px;font-weight:700;color:#1A1A2E;margin-bottom:4px">{{.UnitTitle}}</div>
          <div style="display:inline-block;background:#EEF2FF;color:#3F51B5;font-size:12px;font-weight:600;padding:3px 10px;border-radius:20px;margin-bottom:24px">{{.UnitType}}</div>

          <!-- Message card -->
          <div style="background:{{.MsgBgColor}};border-left:4px solid {{.HeaderColor}};border-radius:0 10px 10px 0;padding:16px 20px;margin-bottom:28px">
            <div style="color:#2D2D2D;font-size:15px;font-weight:500;line-height:1.6">{{.Message}}</div>
          </div>

          <!-- Detail table -->
          <table width="100%" cellpadding="0" cellspacing="0" border="0" style="border-collapse:collapse">
            {{range .Details}}
            <tr>
              <td style="padding:9px 0;border-bottom:1px solid #F0F0F0;color:#888888;font-size:13px;width:130px;vertical-align:top">{{.Label}}</td>
              <td style="padding:9px 0;border-bottom:1px solid #F0F0F0;color:#333333;font-size:13px;font-weight:500;vertical-align:top">{{.Value}}</td>
            </tr>
            {{end}}
          </table>

        </td>
      </tr>

      <!-- ===== FOOTER ===== -->
      <tr>
        <td style="background:#F8F9FA;border-top:1px solid #EEEEEE;padding:20px 32px">
          <table width="100%" cellpadding="0" cellspacing="0" border="0">
            <tr>
              <td style="color:#BBBBBB;font-size:12px">此邮件由 <strong style="color:#999">任务管理器</strong> 自动发送，请勿直接回复</td>
              <td align="right" style="color:#CCCCCC;font-size:12px">运维任务管理平台</td>
            </tr>
          </table>
        </td>
      </tr>

    </table>
  </td></tr>
</table>
</body>
</html>`

const testTmpl = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>任务管理器 邮件测试</title>
</head>
<body style="margin:0;padding:0;background:#F0F2F5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC',sans-serif">
<table width="100%" cellpadding="0" cellspacing="0">
  <tr><td align="center" style="padding:40px 16px">
    <table width="560" cellpadding="0" cellspacing="0" style="max-width:560px;background:#FFF;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,0.10)">
      <tr>
        <td style="background:linear-gradient(135deg,#1565C0,#42A5F5);padding:32px;text-align:center">
          <div style="color:rgba(255,255,255,0.8);font-size:12px;letter-spacing:2px;text-transform:uppercase;margin-bottom:8px">Task Manager</div>
          <div style="color:#FFF;font-size:28px;font-weight:700">✅ 邮件测试成功</div>
        </td>
      </tr>
      <tr>
        <td style="padding:36px;text-align:center">
          <div style="width:72px;height:72px;background:#E8F5E9;border-radius:50%;margin:0 auto 20px;display:flex;align-items:center;justify-content:center;font-size:36px;line-height:72px">📬</div>
          <div style="font-size:17px;font-weight:600;color:#1A1A2E;margin-bottom:10px">SMTP 配置工作正常</div>
          <div style="font-size:14px;color:#666;line-height:1.7">
            您已成功配置任务管理器的邮件通知功能。<br>
            系统将在计时单元触发通知条件时自动向您发送邮件。
          </div>
          <div style="margin-top:24px;background:#F5F5F5;border-radius:8px;padding:12px 20px;display:inline-block;font-size:13px;color:#888">
            发送至：<strong style="color:#333">{{.To}}</strong>
          </div>
        </td>
      </tr>
      <tr>
        <td style="background:#F8F9FA;border-top:1px solid #EEE;padding:18px 32px;text-align:center">
          <div style="color:#BBB;font-size:12px">此邮件由任务管理器自动发送 · 请勿直接回复</div>
        </td>
      </tr>
    </table>
  </td></tr>
</table>
</body>
</html>`

func renderNotification(data NotificationData) (string, error) {
	t, err := template.New("notification").Parse(notificationTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderTest(to string) string {
	t, _ := template.New("test").Parse(testTmpl)
	var buf bytes.Buffer
	_ = t.Execute(&buf, map[string]string{"To": to})
	return buf.String()
}
