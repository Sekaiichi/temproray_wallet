package wallet

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/sekaiichi/temproray_wallet/pkg/types"
)

type testService struct {
	*Service
}

func newTestService() *testService {
	return &testService{Service: &Service{}}
}

type testAccount struct {
	phone    types.Phone
	balance  types.Money
	payments []struct {
		amount   types.Money
		category types.PaymentCategory
	}
}

func (s *testService) addAccount(data testAccount) (*types.Account, []*types.Payment, error) {
	account, err := s.RegisterAccount(data.phone)
	if err != nil {
		return nil, nil, fmt.Errorf("can't register account, error = %v", err)
	}

	err = s.Deposit(account.ID, data.balance)
	if err != nil {
		return nil, nil, fmt.Errorf("can't deposit into account, error = %v", err)
	}
	payments := make([]*types.Payment, len(data.payments))
	for i, payment := range data.payments {
		payments[i], err = s.Pay(account.ID, payment.amount, payment.category)
		if err != nil {
			return nil, nil, fmt.Errorf("can't make payment, erropr = %v", err)
		}
	}
	return account, payments, nil
}

func (s *testService) addAccountWithBalance(phone types.Phone, balance types.Money) (*types.Account, error) {
	account, err := s.RegisterAccount(phone)
	if err != nil {
		return nil, fmt.Errorf("can't register account, error = %v", err)
	}

	err = s.Deposit(account.ID, balance)
	if err != nil {
		return nil, fmt.Errorf("can't deposit account, error = %v", err)
	}

	return account, nil
}

var defaultTestAccount = testAccount{
	phone:   "+992000000001",
	balance: 10_000_00,
	payments: []struct {
		amount   types.Money
		category types.PaymentCategory
	}{
		{amount: 1_000_00, category: "auto"},
	},
}

func TestService_FindAccountByID_success(t *testing.T) {
	svc := &Service{}
	account, err := svc.RegisterAccount("+992000000001")
	if err != nil {
		fmt.Println(err)
		return
	}

	account, err = svc.RegisterAccount("+992000000002")
	if err != nil {
		fmt.Println(err)
		return
	}
	account, err = svc.RegisterAccount("+992000000003")
	if err != nil {
		fmt.Println(err)
		return
	}

	account, err = svc.FindAccountByID(2)
	if err != nil {
		fmt.Println(err)
		return
	}
	var expected = types.Phone("+992000000002")

	if account.Phone != expected {
		t.Errorf("invalid result, expected: %v, got: %v", expected, account.Phone)
	}
}

func TestService_FindAccountByID_notFound(t *testing.T) {
	svc := &Service{}
	account, err := svc.RegisterAccount("+992000000001")
	if err != nil {
		fmt.Println(err)
		return
	}

	account, err = svc.RegisterAccount("+992000000002")
	if err != nil {
		fmt.Println(err)
		return
	}
	account, err = svc.RegisterAccount("+992000000003")
	if err != nil {
		fmt.Println(err)
		return
	}

	account, err = svc.FindAccountByID(4)

	var expected = ErrAccountNotFound

	if err != expected {
		t.Errorf("invalid result, expected: %v, got: %v", expected, account.Phone)
	}
}

func TestService_FindPaymentByID_success(t *testing.T) {
	s := newTestService()

	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	got, err := s.FindPaymentByID(payment.ID)
	if err != nil {
		t.Errorf("FindPaymentByID(): error = %v", err)
		return
	}

	if !reflect.DeepEqual(payment, got) {
		t.Errorf("FindPaymentByID(): wrong payment returned = %v", err)
		return
	}
}

func TestService_FindPaymentByID_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FindPaymentByID(uuid.New().String())
	if err == nil {
		t.Error("FindPaymentByID(): must return error, returned nil")
	}

	if err != ErrPaymentNotFound {
		t.Errorf("FindPaymentByID(): must return ErrPaymentNotFound, returned = %v", err)
		return
	}
}

func TestService_Reject_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	err = s.Reject(payment.ID)
	if err != nil {
		t.Errorf("Reject(): error = %v", err)
		return
	}

	savedPayment, err := s.FindPaymentByID(payment.ID)
	if err != nil {
		t.Errorf("Reject(): can't find payment by id, error = %v", err)
		return
	}

	if savedPayment.Status != types.PaymentStatusFail {
		t.Errorf("Reject(): status didn't change, payment = %v", savedPayment)
		return
	}

	savedAccount, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		t.Errorf("Reject(): balance didn't change, account = %v", savedAccount)
		return
	}
}

func TestService_Repeat_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	_, err = s.Repeat(payment.ID)
	if err != nil {
		t.Errorf("Repeat(): error = %v", err)
		return
	}
}

func TestService_Repeat_fail(t *testing.T) {
	s := newTestService()
	account, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment, err := s.Pay(account.ID, 6_000_00, "auto")

	_, err = s.Repeat(payment.ID)
	if err != ErrNotEnoughBalance {
		t.Errorf("Repeat(): error should be ErrNotEnoughBalance, but got: %v", err)
		return
	}
}

func TestService_FavoritePayment_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	_, err = s.FavoritePayment(payment.ID, "my favorite payment")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestService_FavoritePayment_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FavoritePayment(uuid.New().String(), "my favorite payment")
	if err != ErrPaymentNotFound {
		t.Errorf("FavoritePayment(): error should be ErrPaymentNotFound, but is %v", err)
		return
	}
}

func TestService_PayFromFavorite_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	favorite, err := s.FavoritePayment(payment.ID, "my favorite payment")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.PayFromFavorite(favorite.ID)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestService_PayFromFavorite_fail(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	_, err = s.FavoritePayment(payment.ID, "my favorite payment")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.PayFromFavorite(uuid.New().String())
	if err != ErrFavoriteNotFound {
		t.Errorf("PayFromFavorite(): error should be ErrFavoriteNotFound, but is %v", err)
		return
	}
}

func TestService_ExportToFile(t *testing.T) {
	s := newTestService()
	_, err := s.RegisterAccount("+992000000001")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.RegisterAccount("+992000000002")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.RegisterAccount("+992000000003")
	if err != nil {
		t.Error(err)
		return
	}

	err = s.ExportToFile("export.txt")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestService_ImportFromFile(t *testing.T) {
	s := newTestService()

	err := s.ImportFromFile("export.txt")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestService_Export(t *testing.T) {
	s := newTestService()
	_, err := s.RegisterAccount("+992000000001")
	if err != nil {
		t.Error(err)
		return
	}

	err = s.Deposit(1, 1000000)
	if err != nil {
		t.Error(err)
	}

	_, err = s.Pay(1, 250000, "food")
	if err != nil {
		t.Error(err)
		return
	}

	payment, err := s.Pay(1, 100000, "mobile")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FavoritePayment(payment.ID, "for the phone balance")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.RegisterAccount("+992000000002")
	if err != nil {
		t.Error(err)
		return
	}

	err = s.Deposit(2, 2000000)
	if err != nil {
		t.Error(err)
	}

	_, err = s.Pay(2, 750000, "mobile")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.RegisterAccount("+992000000003")
	if err != nil {
		t.Error(err)
		return
	}

	err = s.Deposit(3, 3000000)
	if err != nil {
		t.Error(err)
	}

	_, err = s.Pay(3, 1250000, "auto")
	if err != nil {
		t.Error(err)
		return
	}

	payment, err = s.Pay(3, 150000, "food")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FavoritePayment(payment.ID, "Iskandar-Kebap")
	if err != nil {
		t.Error(err)
		return
	}

	payment, err = s.Pay(3, 250000, "mobile")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FavoritePayment(payment.ID, "Закинуть на баланс")
	if err != nil {
		t.Error(err)
		return
	}

	err = s.Export("data")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestService_Import(t *testing.T) {
	s := newTestService()
	err := s.Import("data")
	if err != nil {
		t.Error(err)
	}
}

func fillData(s *testService) {
	s.RegisterAccount("+992000000001")
	s.Deposit(1, 10_000_00)
	s.Pay(1, 1, "food")
	s.Pay(1, 2, "mobile")
	s.Pay(1, 3, "transport")
	s.Pay(1, 4, "mobile")
	s.Pay(1, 5, "food")
	s.Pay(1, 6, "auto")
	s.Pay(1, 7, "bank")

	s.RegisterAccount("+992000000002")
	s.Deposit(2, 2000000)
	s.Pay(2, 8, "mobile")

	s.RegisterAccount("+992000000003")
	s.Deposit(3, 3000000)
	s.Pay(3, 9, "auto")
	s.Pay(3, 10, "food")
	s.Pay(3, 11, "mobile")
}

func TestService_ExportAccountHistory(t *testing.T) {
	s := newTestService()
	fillData(s)

	_, err := s.ExportAccountHistory(1)
	if err != nil {
		t.Error(err)
	}
}

func TestService_HistoryToFiles(t *testing.T) {
	s := newTestService()
	fillData(s)

	payments, err := s.ExportAccountHistory(1)
	if err != nil {
		t.Error(err)
	}

	err = s.HistoryToFiles(payments, "history", 3)
	if err != nil {
		t.Error(err)
	}
}

func TestService_SumPayments(t *testing.T) {
	s := newTestService()
	fillData(s)

	sum := s.SumPayments(3)

	if sum != 66 {
		t.Error(sum)
	}
}

func BenchmarkSumPayments(b *testing.B) {
	s := newTestService()
	fillData(s)

	b.ResetTimer()

	want := types.Money(66)
	for i := 0; i < b.N; i++ {
		result := s.SumPayments(2)
		b.StopTimer()
		if result != want {
			b.Fatalf("Invalid result, got %v, want %v", result, want)
		}
		b.StartTimer()
	}
}

func TestService_FilterPayments(t *testing.T) {
	s := newTestService()
	fillData(s)

	payments, err := s.FilterPayments(1, 2)
	if err != nil {
		t.Error(err)
		t.Error(payments)
	}
}

func BenchmarkFilterPayments(b *testing.B) {
	s := newTestService()
	fillData(s)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := s.FilterPayments(1, 2)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func TestService_FilterPaymentsByFn(t *testing.T) {
	s := newTestService()
	fillData(s)

	payments, err := s.FilterPaymentsByFn(FilterMobile, 2)
	if err != nil {
		t.Error(err)
		t.Error(payments)
	}
}

func BenchmarkFilterPaymentsByFn(b *testing.B) {
	s := newTestService()
	fillData(s)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := s.FilterPaymentsByFn(FilterMobile, 2)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func TestService_SumPaymentsWithProgress(t *testing.T) {
	s := newTestService()

	for i := 0; i < 1_000_000; i++ {
		paymentID := uuid.New().String()
		payment := &types.Payment{
			ID:        paymentID,
			AccountID: int64(i % 10),
			Amount:    types.Money(i),
			Category:  "test",
			Status:    types.PaymentStatusInProgress,
		}
		s.payments = append(s.payments, payment)
	}

	sum := types.Money(0)

	for i := range s.SumPaymentsWithProgress() {
		sum += i.Result
	}

	if sum != 499999500000 {
		t.Error("Error")
	}
}

func BenchmarkSumPaymentsWithProgress(b *testing.B) {
	s := newTestService()
	for i := 0; i < 1_000_000; i++ {
		paymentID := uuid.New().String()
		payment := &types.Payment{
			ID:        paymentID,
			AccountID: int64(i % 10),
			Amount:    types.Money(i),
			Category:  "test",
			Status:    types.PaymentStatusInProgress,
		}
		s.payments = append(s.payments, payment)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sum := types.Money(0)
		for j := range s.SumPaymentsWithProgress() {
			sum += j.Result
		}
		b.StopTimer()

		if sum != 499999500000 {
			b.Fatal(i, sum)
		}
		b.StartTimer()
	}
}