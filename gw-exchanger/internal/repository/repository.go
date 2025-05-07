package repository

type ExchangeRepo interface {
	GetRates() (map[string]float64, error)
	Exchange(string, string) (float64, error)
}
