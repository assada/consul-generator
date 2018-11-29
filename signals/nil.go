package signals

type NilSignal int

func (s *NilSignal) String() string { return "SIGNIL" }
func (s *NilSignal) Signal()        {}
