package core

// Component là đơn vị chạy của Fluxor (tương đương Verticle)
type Component interface {
	OnStart(ctx *FluxorContext) error
	OnStop() error
}
