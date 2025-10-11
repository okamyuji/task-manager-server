package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"

	"github.com/resend/resend-go/v2"
)

// EmailSender メール送信インターフェース
type EmailSender interface {
	SendVerificationCode(ctx context.Context, to, name, code string) error
	SendWelcomeEmail(ctx context.Context, to, name string) error
}

// EmailService メール送信サービス
type EmailService struct {
	resendClient *resend.Client // Resendクライアント（本番環境用）
	smtpHost     string         // SMTPホスト（開発環境用）
	smtpPort     string         // SMTPポート（開発環境用）
	from         string         // 送信元メールアドレス
	logger       *slog.Logger
}

// NewEmailService メールサービスを作成
func NewEmailService(logger *slog.Logger) *EmailService {
	// Resend APIキーの確認（本番環境用）
	resendAPIKey := os.Getenv("RESEND_API_KEY")
	var resendClient *resend.Client
	if resendAPIKey != "" {
		resendClient = resend.NewClient(resendAPIKey)
		logger.Info("メール送信: Resend API を使用します")
	} else {
		logger.Info("メール送信: SMTP (MailHog) を使用します")
	}

	// SMTP設定（開発環境用）
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "mailhog" // Docker Compose内のデフォルト
	}

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1025" // MailHogのデフォルトポート
	}

	// 送信元メールアドレス
	from := os.Getenv("MAIL_FROM")
	if from == "" {
		if resendAPIKey != "" {
			// 本番環境ではドメイン認証済みのアドレスが必要
			from = "noreply@yourdomain.com"
		} else {
			// 開発環境（MailHog）
			from = "noreply@taskmanager.local"
		}
	}

	return &EmailService{
		resendClient: resendClient,
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		from:         from,
		logger:       logger,
	}
}

// SendVerificationCode 認証コードを送信
func (s *EmailService) SendVerificationCode(ctx context.Context, to, name, code string) error {
	subject := "【Task Manager】認証コードのお知らせ"
	body := s.buildVerificationEmailBody(name, code)

	return s.sendEmail(to, subject, body)
}

// SendWelcomeEmail ウェルカムメールを送信
func (s *EmailService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	subject := "【Task Manager】ご登録ありがとうございます"
	body := s.buildWelcomeEmailBody(name)

	return s.sendEmail(to, subject, body)
}

// sendEmail メールを送信（環境に応じて自動切り替え）
func (s *EmailService) sendEmail(to, subject, body string) error {
	// 本番環境: Resend API
	if s.resendClient != nil {
		return s.sendViaResend(to, subject, body)
	}

	// 開発環境: SMTP（MailHog）
	return s.sendViaSMTP(to, subject, body)
}

// sendViaResend Resend APIでメール送信（本番環境）
func (s *EmailService) sendViaResend(to, subject, body string) error {
	s.logger.Info("Resend APIでメール送信中",
		"to", to,
		"subject", subject,
	)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Text:    body, // プレーンテキスト
	}

	sent, err := s.resendClient.Emails.Send(params)
	if err != nil {
		s.logger.Error("Resend APIメール送信失敗",
			"error", err,
			"to", to,
		)
		return fmt.Errorf("resendメール送信失敗: %w", err)
	}

	s.logger.Info("Resend APIメール送信成功",
		"to", to,
		"message_id", sent.Id,
	)
	return nil
}

// sendViaSMTP SMTPでメール送信（開発環境：MailHog）
func (s *EmailService) sendViaSMTP(to, subject, body string) error {
	// メール形式
	message := s.buildMessage(s.from, to, subject, body)

	// SMTPサーバーに接続（認証なし: MailHog用）
	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)

	s.logger.Info("SMTP経由でメール送信中",
		"to", to,
		"subject", subject,
		"smtp", addr,
	)

	// MailHogは認証不要のため、smtp.SendMailを直接使用
	err := smtp.SendMail(
		addr,
		nil, // 認証なし
		s.from,
		[]string{to},
		[]byte(message),
	)

	if err != nil {
		s.logger.Error("SMTPメール送信失敗",
			"error", err,
			"to", to,
			"smtp", addr,
		)
		return fmt.Errorf("SMTPメール送信失敗: %w", err)
	}

	s.logger.Info("SMTPメール送信成功", "to", to)
	return nil
}

// buildMessage メールメッセージを構築（SMTP用）
func (s *EmailService) buildMessage(from, to, subject, body string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("From: %s\r\n", from))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(body)

	return builder.String()
}

// buildVerificationEmailBody 認証コードメール本文を構築
func (s *EmailService) buildVerificationEmailBody(name, code string) string {
	return fmt.Sprintf(`%s 様

Task Managerにご登録いただきありがとうございます。

以下の認証コードを入力して、アカウント登録を完了してください。

━━━━━━━━━━━━━━━━━━━━
認証コード: %s
━━━━━━━━━━━━━━━━━━━━

※このコードの有効期限は15分です。
※このメールに心当たりがない場合は、無視してください。

Task Manager運営チーム
`, name, code)
}

// buildWelcomeEmailBody ウェルカムメール本文を構築
func (s *EmailService) buildWelcomeEmailBody(name string) string {
	return fmt.Sprintf(`%s 様

Task Managerへようこそ！

アカウント認証が完了しました。
今すぐタスク管理を始めましょう。

【主な機能】
・タスクの作成・編集・削除
・優先度とタグによる整理
・期限管理
・画像添付

ご不明な点がございましたら、お気軽にお問い合わせください。

Task Manager運営チーム
`, name)
}
