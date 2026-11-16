package mail

import (
	"fmt"
	"strings"
)

// Localized transactional templates (WI-A2): password reset + email
// verification render in the recipient's users.locale. ko is the original,
// en is hand-written, ja/zh/th are machine drafts (Beta, same as web i18n).

// NormalizeLocale maps a user locale code to a template locale
// (ko|en|ja|zh|th). Unknown or empty codes fall back to ko, the service
// default, so a bad stored value never blocks mail delivery.
func NormalizeLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en":
		return "en"
	case "ja":
		return "ja"
	case "zh", "zh-cn", "zh_cn":
		return "zh"
	case "th":
		return "th"
	default:
		return "ko"
	}
}

type verifyTemplate struct {
	subject string
	body    string // Printf verbs: %d ttl hours, %s link.
}

var verifyTemplates = map[string]verifyTemplate{
	"ko": {
		subject: "[AVIBE] 이메일 인증",
		body: "AVIBE 이메일 인증 링크입니다 (%d시간 유효, 1회용):\n%s" +
			"\n\n본인이 요청하지 않았다면 이 메일을 무시하세요.",
	},
	"en": {
		subject: "[AVIBE] Verify your email",
		body: "Here is your AVIBE email verification link (valid for %d hours, single use):\n%s" +
			"\n\nIf you didn't request this, ignore this email.",
	},
	"ja": {
		subject: "[AVIBE] メールアドレスの確認",
		body: "AVIBEのメール確認リンクです（%d時間有効、1回限り）:\n%s" +
			"\n\n身に覚えがない場合は、このメールを無視してください。",
	},
	"zh": {
		subject: "[AVIBE] 验证您的电子邮箱",
		body: "这是您的 AVIBE 邮箱验证链接（%d小时内有效，仅可使用一次）:\n%s" +
			"\n\n如果这不是您本人的操作，请忽略此邮件。",
	},
	"th": {
		subject: "[AVIBE] ยืนยันอีเมลของคุณ",
		body: "นี่คือลิงก์ยืนยันอีเมล AVIBE ของคุณ (ใช้ได้ %d ชั่วโมง ใช้ได้ครั้งเดียว):\n%s" +
			"\n\nหากคุณไม่ได้เป็นผู้ขอนี้ โปรดละเว้นอีเมลฉบับนี้",
	},
}

type resetTemplate struct {
	subject string
	body    string // Printf verbs: %d ttl minutes, %s link.
}

var resetTemplates = map[string]resetTemplate{
	"ko": {
		subject: "[AVIBE] 비밀번호 재설정",
		body: "AVIBE 비밀번호 재설정 링크입니다 (%d분 유효, 1회용):\n%s" +
			"\n\n본인이 요청하지 않았다면 이 메일을 무시하세요.",
	},
	"en": {
		subject: "[AVIBE] Reset your password",
		body: "Here is your AVIBE password reset link (valid for %d minutes, single use):\n%s" +
			"\n\nIf you didn't request this, ignore this email.",
	},
	"ja": {
		subject: "[AVIBE] パスワードの再設定",
		body: "AVIBEのパスワード再設定リンクです（%d分有効、1回限り）:\n%s" +
			"\n\n身に覚えがない場合は、このメールを無視してください。",
	},
	"zh": {
		subject: "[AVIBE] 重置您的密码",
		body: "这是您的 AVIBE 密码重置链接（%d分钟内有效，仅可使用一次）:\n%s" +
			"\n\n如果这不是您本人的操作，请忽略此邮件。",
	},
	"th": {
		subject: "[AVIBE] รีเซ็ตรหัสผ่านของคุณ",
		body: "นี่คือลิงก์รีเซ็ตรหัสผ่าน AVIBE ของคุณ (ใช้ได้ %d นาที ใช้ได้ครั้งเดียว):\n%s" +
			"\n\nหากคุณไม่ได้เป็นผู้ขอนี้ โปรดละเว้นอีเมลฉบับนี้",
	},
}

// RenderVerify returns the localized verification subject + text body.
func RenderVerify(locale, link string, ttlHours int) (subject, body string) {
	t := verifyTemplates[NormalizeLocale(locale)]
	return t.subject, fmt.Sprintf(t.body, ttlHours, link)
}

// RenderReset returns the localized password-reset subject + text body.
func RenderReset(locale, link string, ttlMinutes int) (subject, body string) {
	t := resetTemplates[NormalizeLocale(locale)]
	return t.subject, fmt.Sprintf(t.body, ttlMinutes, link)
}
