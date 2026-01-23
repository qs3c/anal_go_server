package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/qs3c/anal_go_server/config"
)

type Service struct {
	cfg *config.EmailConfig
}

func NewService(cfg *config.EmailConfig) *Service {
	return &Service{cfg: cfg}
}

// SendVerificationCode 发送邮箱验证码
func (s *Service) SendVerificationCode(to, code string) error {
	subject := "验证码 - Go 项目结构分析平台"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">邮箱验证</h2>
        <p>您好，</p>
        <p>您正在注册 Go 项目结构分析平台账号，验证码为：</p>
        <div style="background-color: #f3f4f6; padding: 15px; text-align: center; font-size: 24px; font-weight: bold; letter-spacing: 5px; margin: 20px 0;">
            %s
        </div>
        <p>验证码有效期为 10 分钟，请尽快完成验证。</p>
        <p>如果您没有进行此操作，请忽略此邮件。</p>
        <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
        <p style="color: #6b7280; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
    </div>
</body>
</html>
`, code)

	return s.sendHTML(to, subject, body)
}

// SendPasswordReset 发送密码重置邮件
func (s *Service) SendPasswordReset(to, resetLink string) error {
	subject := "密码重置 - Go 项目结构分析平台"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">密码重置</h2>
        <p>您好，</p>
        <p>您正在请求重置密码，请点击下方按钮完成重置：</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #2563eb; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">重置密码</a>
        </div>
        <p>或者复制以下链接到浏览器：</p>
        <p style="background-color: #f3f4f6; padding: 10px; word-break: break-all;">%s</p>
        <p>链接有效期为 30 分钟。</p>
        <p>如果您没有请求重置密码，请忽略此邮件。</p>
        <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
        <p style="color: #6b7280; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
    </div>
</body>
</html>
`, resetLink, resetLink)

	return s.sendHTML(to, subject, body)
}

// SendWelcome 发送欢迎邮件
func (s *Service) SendWelcome(to, username string) error {
	subject := "欢迎加入 - Go 项目结构分析平台"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">欢迎加入！</h2>
        <p>您好，%s！</p>
        <p>感谢您注册 Go 项目结构分析平台。</p>
        <p>现在您可以：</p>
        <ul>
            <li>分析 Go 项目的结构体依赖关系</li>
            <li>生成可视化框图</li>
            <li>与社区分享您的分析</li>
        </ul>
        <p>开始探索吧！</p>
        <hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
        <p style="color: #6b7280; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
    </div>
</body>
</html>
`, username)

	return s.sendHTML(to, subject, body)
}

// sendHTML 发送 HTML 邮件
func (s *Service) sendHTML(to, subject, body string) error {
	headers := make(map[string]string)
	headers["From"] = s.cfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg.String()))
}

// sendPlain 发送纯文本邮件
func (s *Service) sendPlain(to, subject, body string) error {
	headers := make(map[string]string)
	headers["From"] = s.cfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg.String()))
}
