package email

type Provider interface {
	Send(to, subject, body string) error
}

type Message struct {
	To      string
	Subject string
	Body    string
}
