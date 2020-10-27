package types

//Money describes amount of money in minimal values (cents)
type Money int64

//PaymentCategory describes the category in which the payments are made
type PaymentCategory string

//PaymentStatus describes the status of payment
type PaymentStatus string

//Status codes
const (
	PaymentStatusOk         PaymentStatus = "OK"
	PaymentStatusFail       PaymentStatus = "FAIL"
	PaymentStatusInProgress PaymentStatus = "INPROGRESS"
)

//Payment describes the payment information
type Payment struct {
	ID        string
	AccountID int64
	Amount    Money
	Category  PaymentCategory
	Status    PaymentStatus
}

//Phone describes the phone number
type Phone string

//Account describes the user account
type Account struct {
	ID      int64
	Phone   Phone
	Balance Money
}

//Favorite holds the info abouth favorite payments
type Favorite struct {
	ID        string
	AccountID int64
	Name      string
	Amount    Money
	Category  PaymentCategory
}
