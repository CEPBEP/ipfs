package providers

type Stat struct {
	ProvideBufLen int
}

func (p *providers) Stat() (*Stat, error) {
	return &Stat{
		ProvideBufLen: len(p.newBlocks),
	}, nil
}
