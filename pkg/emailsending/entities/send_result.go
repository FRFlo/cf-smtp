package entities

type SendResult struct {
	Delivered        []string
	Queued           []string
	PermanentBounces []string
}
