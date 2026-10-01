// Package mailer 负责渲染与发送系统邮件（PRD DLV-08）。
// 业务代码通过 Queue 投递异步任务，由 worker 调用 SMTPSender 实际发送，失败自动重试。
package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/wneessen/go-mail"
	"go.uber.org/zap"

	"mkclou/server/internal/pkg/config"
)

// Message 是一封待发送的邮件。
type Message struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTML     string `json:"html"`
	Text     string `json:"text"`
	Template string `json:"template"` // 模板名，仅用于日志与统计
}

// Sender 发送邮件。
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// SMTPSender 通过 SMTP 发送邮件。
type SMTPSender struct {
	cfg config.MailConfig
}

func NewSMTPSender(cfg config.MailConfig) *SMTPSender { return &SMTPSender{cfg: cfg} }

func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	m := mail.NewMsg()
	if err := m.From(s.cfg.From); err != nil {
		return fmt.Errorf("from: %w", err)
	}
	if err := m.To(msg.To); err != nil {
		return fmt.Errorf("to: %w", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(mail.TypeTextPlain, msg.Text)
	m.AddAlternativeString(mail.TypeTextHTML, msg.HTML)

	opts := []mail.Option{mail.WithPort(s.cfg.Port), mail.WithTimeout(15 * time.Second)}
	if s.cfg.TLS {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}
	if s.cfg.Username != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithUsername(s.cfg.Username), mail.WithPassword(s.cfg.Password))
	}
	client, err := mail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	return client.DialAndSendWithContext(ctx, m)
}

// TypeSendEmail 是发送邮件的 Asynq 任务类型。
const TypeSendEmail = "email:send"

// Queue 把邮件投递为异步任务。
type Queue interface {
	Enqueue(ctx context.Context, msg Message) error
}

type AsynqQueue struct {
	client *asynq.Client
}

func NewAsynqQueue(client *asynq.Client) *AsynqQueue { return &AsynqQueue{client: client} }

// Enqueue 投递邮件任务：失败最多重试 3 次（PRD 总览 5.5），任务结果保留 1 天便于排查。
func (q *AsynqQueue) Enqueue(ctx context.Context, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(TypeSendEmail, payload),
		asynq.Queue("default"), asynq.MaxRetry(3), asynq.Timeout(30*time.Second), asynq.Retention(24*time.Hour))
	return err
}

// RetryDelay 按 1 分钟、5 分钟、30 分钟的间隔重试。
func RetryDelay(n int, _ error, task *asynq.Task) time.Duration {
	if task.Type() != TypeSendEmail {
		return asynq.DefaultRetryDelayFunc(n, nil, task)
	}
	switch n {
	case 0, 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}

// NewHandler 返回 worker 端的任务处理函数。日志中不记录邮件正文（可能包含令牌、卡密）。
func NewHandler(sender Sender, log *zap.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var msg Message
		if err := json.Unmarshal(t.Payload(), &msg); err != nil {
			return fmt.Errorf("decode payload: %w: %w", err, asynq.SkipRetry)
		}
		if err := sender.Send(ctx, msg); err != nil {
			log.Warn("send email failed", zap.String("template", msg.Template), zap.Error(err))
			return err
		}
		log.Info("email sent", zap.String("template", msg.Template))
		return nil
	}
}
