package rabbit

type pool chan struct{}

func (p pool) Acquire() {
	p <- struct{}{}
}

func (p pool) Release() {
	<-p
}

func (p pool) Wait() {
	for i := 0; i < cap(p); i++ {
		p <- struct{}{}
	}
}
