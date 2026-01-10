package events

var GlobalProducer Producer

func SetGlobalProducer(p Producer) {
	GlobalProducer = p
}

func GetGlobalProducer() Producer {
	return GlobalProducer
}
