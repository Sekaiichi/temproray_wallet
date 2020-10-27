package wallet

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/sekaiichi/temproray_wallet/pkg/types"
)

//ErrPhoneRegistered error for phone already registered
var ErrPhoneRegistered = errors.New("phone already registered")

//ErrAmountMustBePositive error for less than zero
var ErrAmountMustBePositive = errors.New("amount must be greater than zero")

//ErrAccountNotFound error for account not found
var ErrAccountNotFound = errors.New("account not found")

//ErrNotEnoughBalance error for balance less than amount
var ErrNotEnoughBalance = errors.New("not enough balance")

//ErrPaymentNotFound error for inexistent payment
var ErrPaymentNotFound = errors.New("payment not found")

//ErrFavoriteNotFound error for inexistent payment
var ErrFavoriteNotFound = errors.New("favorite not found")

//Service holds the slices of all the payments and all user accounts
type Service struct {
	nextAccountID int64
	accounts      []*types.Account
	payments      []*types.Payment
	favorites     []*types.Favorite
}

//RegisterAccount method searches for an existing phone number, and if none found - creates an account
func (s *Service) RegisterAccount(phone types.Phone) (*types.Account, error) {
	for _, account := range s.accounts {
		if account.Phone == phone {
			return nil, ErrPhoneRegistered
		}
	}
	s.nextAccountID++
	account := &types.Account{
		ID:      s.nextAccountID,
		Phone:   phone,
		Balance: 0,
	}

	s.accounts = append(s.accounts, account)

	return account, nil
}

//Deposit method increases the account balance by amount
func (s *Service) Deposit(accountID int64, amount types.Money) error {
	if amount <= 0 {
		return ErrAmountMustBePositive
	}

	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return ErrAccountNotFound
	}

	account.Balance += amount
	return nil
}

//Pay returns payment struct, while decreasing the amount from account balance
func (s *Service) Pay(accountID int64, amount types.Money, category types.PaymentCategory) (*types.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return nil, ErrAccountNotFound
	}

	if account.Balance < amount {
		return nil, ErrNotEnoughBalance
	}

	account.Balance -= amount
	paymentID := uuid.New().String()
	payment := &types.Payment{
		ID:        paymentID,
		AccountID: accountID,
		Amount:    amount,
		Category:  category,
		Status:    types.PaymentStatusInProgress,
	}

	s.payments = append(s.payments, payment)
	return payment, nil
}

//FindAccountByID returns the pointer to an account and an error
func (s *Service) FindAccountByID(accountID int64) (*types.Account, error) {
	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}

	return account, nil
}

//Reject rejects the payment
func (s *Service) Reject(paymentID string) error {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return err
	}

	account, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		return err
	}

	payment.Status = types.PaymentStatusFail
	account.Balance += payment.Amount
	return nil
}

//FindPaymentByID returns the pointer to a payment and an error
func (s *Service) FindPaymentByID(paymentID string) (*types.Payment, error) {
	for _, payment := range s.payments {
		if payment.ID == paymentID {
			return payment, nil
		}
	}
	return nil, ErrPaymentNotFound
}

//Repeat repeats the payment
func (s *Service) Repeat(paymentID string) (*types.Payment, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	account, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		return nil, err
	}

	if payment.Amount > account.Balance {
		return nil, ErrNotEnoughBalance
	}

	newPayment, err := s.Pay(account.ID, payment.Amount, payment.Category)
	if err != nil {
		return nil, err
	}

	return newPayment, nil
}

//FavoritePayment adds a payment to the favorites
func (s *Service) FavoritePayment(paymentID string, name string) (*types.Favorite, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	favorite := &types.Favorite{
		ID:        uuid.New().String(),
		AccountID: payment.AccountID,
		Name:      name,
		Amount:    payment.Amount,
		Category:  payment.Category,
	}

	s.favorites = append(s.favorites, favorite)
	return favorite, nil
}

//FindFavoriteByID returns the pointer to a payment and an error
func (s *Service) FindFavoriteByID(favoriteID string) (*types.Favorite, error) {
	for _, favorite := range s.favorites {
		if favorite.ID == favoriteID {
			return favorite, nil
		}
	}
	return nil, ErrFavoriteNotFound
}

//PayFromFavorite makes payment from favorite list
func (s *Service) PayFromFavorite(favoriteID string) (*types.Payment, error) {
	favorite, err := s.FindFavoriteByID(favoriteID)
	if err != nil {
		return nil, err
	}

	account, err := s.FindAccountByID(favorite.AccountID)
	if err != nil {
		return nil, err
	}

	if favorite.Amount > account.Balance {
		return nil, ErrNotEnoughBalance
	}

	payment, err := s.Pay(account.ID, favorite.Amount, favorite.Category)
	if err != nil {
		return nil, err
	}
	return payment, nil
}

//ExportToFile writes the data into a file
func (s *Service) ExportToFile(path string) error {
	records := make([]byte, 0)
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Print(cerr)
		}
	}()

	for _, account := range s.accounts {
		buffer := make([]byte, 0)
		buffer = strconv.AppendInt(buffer, account.ID, 10)
		buffer = append(buffer, ";"...)
		buffer = append(buffer, string(account.Phone)...)
		buffer = append(buffer, ";"...)
		buffer = strconv.AppendInt(buffer, int64(account.Balance), 10)
		buffer = append(buffer, "|"...)
		records = append(records, buffer...)
	}

	_, werr := file.Write(records)
	if err != nil {
		log.Print(werr)
		return werr
	}
	return nil
}

//ImportFromFile writes the data into a file
func (s *Service) ImportFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Print(cerr)
		}
	}()

	content := make([]byte, 0)
	buffer := make([]byte, 4)
	for {
		read, err := file.Read(buffer)
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		content = append(content, buffer[:read]...)
	}

	data := string(content)
	records := strings.Split(data, "|")

	if records[len(records)-1] == "" {
		records = records[:len(records)-1] //truncate if the last item after splitting by "|" is empty
	}

	for _, record := range records {
		fields := strings.Split(record, ";")

		id, _ := strconv.Atoi(fields[0])
		phone := types.Phone(fields[1])
		balance, _ := strconv.Atoi(fields[2])

		account := &types.Account{
			ID:      int64(id),
			Phone:   phone,
			Balance: types.Money(balance),
		}
		s.accounts = append(s.accounts, account)
	}
	return nil
}

//Export method exports the data into corresponding dump files
func (s *Service) Export(dir string) error {

	_, werr := os.Stat(dir)
	if os.IsNotExist(werr) {
		werr = os.Mkdir(dir, 0777)
	}
	if werr != nil {
		return werr
	}

	if len(s.accounts) != 0 {
		buffer := make([]byte, 0)
		for _, account := range s.accounts {
			buffer = strconv.AppendInt(buffer, account.ID, 10)
			buffer = append(buffer, ';')
			buffer = append(buffer, account.Phone...)
			buffer = append(buffer, ';')
			buffer = strconv.AppendInt(buffer, int64(account.Balance), 10)
			buffer = append(buffer, '\n')
		}

		werr = ioutil.WriteFile(dir+"/accounts.dump", buffer, 0777)
		if werr != nil {
			return werr
		}
	}

	if len(s.payments) != 0 {
		buffer := make([]byte, 0)
		for _, payment := range s.payments {
			buffer = append(buffer, payment.ID...)
			buffer = append(buffer, ';')
			buffer = strconv.AppendInt(buffer, payment.AccountID, 10)
			buffer = append(buffer, ';')
			buffer = strconv.AppendInt(buffer, int64(payment.Amount), 10)
			buffer = append(buffer, ';')
			buffer = append(buffer, payment.Category...)
			buffer = append(buffer, ';')
			buffer = append(buffer, payment.Status...)
			buffer = append(buffer, '\n')
		}

		werr = ioutil.WriteFile(dir+"/payments.dump", buffer, 0777)
		if werr != nil {
			return werr
		}
	}

	if len(s.favorites) != 0 {
		buffer := make([]byte, 0)
		for _, favorite := range s.favorites {
			buffer = append(buffer, favorite.ID...)
			buffer = append(buffer, ';')
			buffer = strconv.AppendInt(buffer, favorite.AccountID, 10)
			buffer = append(buffer, ';')
			buffer = append(buffer, favorite.Name...)
			buffer = append(buffer, ';')
			buffer = strconv.AppendInt(buffer, int64(favorite.Amount), 10)
			buffer = append(buffer, ';')
			buffer = append(buffer, favorite.Category...)
			buffer = append(buffer, '\n')
		}

		werr = ioutil.WriteFile(dir+"/favorites.dump", buffer, 0777)
		if werr != nil {
			return werr
		}
	}
	return nil
}

//Import method imports the data from specified directory
func (s *Service) Import(dir string) error {
	accountsExist := false
	paymentsExist := false
	favoritesExist := false

	_, rerr := os.Stat(dir)
	if rerr != nil {
		return rerr
	}

	folder, rerr := os.Open(dir + "/.")
	if rerr != nil {
		return rerr
	}
	defer folder.Close()

	list, rerr := folder.Readdirnames(0) // 0 to read all files and folders
	if rerr != nil {
		return rerr
	}

	for _, name := range list {
		if name == "accounts.dump" {
			accountsExist = true
		}
		if name == "payments.dump" {
			paymentsExist = true
		}
		if name == "favorites.dump" {
			favoritesExist = true
		}
	}

	if accountsExist {
		content := make([]byte, 0)
		content, rerr = ioutil.ReadFile(dir + "/accounts.dump")
		if rerr != nil {
			return rerr
		}

		data := string(content)
		records := strings.Split(data, "\n")

		if records[len(records)-1] == "" {
			records = records[:len(records)-1] //truncate if the last record after splitting by "\n" is empty
		}

		for _, record := range records {
			fields := strings.Split(record, ";")

			accountID, _ := strconv.Atoi(fields[0])
			accountPhone := types.Phone(fields[1])
			accountBalance, _ := strconv.Atoi(fields[2])

			var account *types.Account
			for _, acc := range s.accounts {
				if acc.ID == int64(accountID) {
					account = acc
					break
				}
			}

			if account == nil {
				newAccount := &types.Account{
					ID:      int64(accountID),
					Phone:   accountPhone,
					Balance: types.Money(accountBalance),
				}
				s.accounts = append(s.accounts, newAccount)
			} else {
				account.Phone = accountPhone
				account.Balance = types.Money(accountBalance)
			}
		}

		var max = int64(0)
		for _, acc := range s.accounts {
			if acc.ID > max {
				max = acc.ID
			}
		}
		s.nextAccountID = max
	}

	if paymentsExist {
		content := make([]byte, 0)
		content, rerr = ioutil.ReadFile(dir + "/payments.dump")
		if rerr != nil {
			return rerr
		}

		data := string(content)
		records := strings.Split(data, "\n")

		if records[len(records)-1] == "" {
			records = records[:len(records)-1] //truncate if the last record after splitting by "\n" is empty
		}

		for _, record := range records {
			fields := strings.Split(record, ";")

			paymentID := fields[0]
			paymentAccountID, _ := strconv.Atoi(fields[1])
			paymentAmount, _ := strconv.Atoi(fields[2])
			paymentCategory := fields[3]
			paymentStatus := fields[4]

			var payment *types.Payment
			for _, paymentIterator := range s.payments {
				if paymentIterator.ID == paymentID {
					payment = paymentIterator
					break
				}
			}

			if payment == nil {
				newPayment := &types.Payment{
					ID:        paymentID,
					AccountID: int64(paymentAccountID),
					Amount:    types.Money(paymentAmount),
					Category:  types.PaymentCategory(paymentCategory),
					Status:    types.PaymentStatus(paymentStatus),
				}
				s.payments = append(s.payments, newPayment)
			} else {
				payment.AccountID = int64(paymentAccountID)
				payment.Amount = types.Money(paymentAmount)
				payment.Category = types.PaymentCategory(paymentCategory)
				payment.Status = types.PaymentStatus(paymentStatus)
			}
		}
	}

	if favoritesExist {
		content := make([]byte, 0)
		content, rerr = ioutil.ReadFile(dir + "/favorites.dump")
		if rerr != nil {
			return rerr
		}

		data := string(content)
		records := strings.Split(data, "\n")

		if records[len(records)-1] == "" {
			records = records[:len(records)-1] //truncate if the last record after splitting by "\n" is empty
		}

		for _, record := range records {
			fields := strings.Split(record, ";")

			favoriteID := fields[0]
			favoriteAccountID, _ := strconv.Atoi(fields[1])
			favoriteName := fields[2]
			favoriteAmount, _ := strconv.Atoi(fields[3])
			favoriteCategory := fields[4]

			var favorite *types.Favorite
			for _, fav := range s.favorites {
				if fav.ID == favoriteID {
					favorite = fav
					break
				}
			}

			if favorite == nil {
				newFavorite := &types.Favorite{
					ID:        favoriteID,
					AccountID: int64(favoriteAccountID),
					Name:      favoriteName,
					Amount:    types.Money(favoriteAmount),
					Category:  types.PaymentCategory(favoriteCategory),
				}
				s.favorites = append(s.favorites, newFavorite)
			} else {
				favorite.AccountID = int64(favoriteAccountID)
				favorite.Name = favoriteName
				favorite.Amount = types.Money(favoriteAmount)
				favorite.Category = types.PaymentCategory(favoriteCategory)
			}
		}
	}
	return nil
}

//ExportAccountHistory method copies all payments of a given accountID into a new slice
func (s *Service) ExportAccountHistory(accountID int64) ([]types.Payment, error) {
	payments := make([]types.Payment, 0)

	_, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	for _, payment := range s.payments {
		if payment.AccountID == accountID {
			payments = append(payments, *payment)
		}
	}

	return payments, nil
}

//HistoryToFiles method exports given payments slice into a {payments[n].dump} files in {dir} directory, each containing {records} items
func (s *Service) HistoryToFiles(payments []types.Payment, dir string, records int) error {

	_, werr := os.Stat(dir)
	if os.IsNotExist(werr) {
		werr = os.Mkdir(dir, 0777)
	}
	if werr != nil {
		return werr
	}

	buffer := make([]byte, 0)

	for i, payment := range payments {

		buffer = append(buffer, payment.ID...)
		buffer = append(buffer, ';')
		buffer = strconv.AppendInt(buffer, payment.AccountID, 10)
		buffer = append(buffer, ';')
		buffer = strconv.AppendInt(buffer, int64(payment.Amount), 10)
		buffer = append(buffer, ';')
		buffer = append(buffer, payment.Category...)
		buffer = append(buffer, ';')
		buffer = append(buffer, payment.Status...)
		buffer = append(buffer, '\n')

		if len(payments) <= records {
			file := dir + "/payments.dump"
			werr := ioutil.WriteFile(file, buffer, 0777)
			if werr != nil {
				return werr
			}
		} else if (i+1)%records == 0 || i == len(payments)-1 {
			file := dir + "/payments" + strconv.Itoa((i/records)+1) + ".dump"
			werr := ioutil.WriteFile(file, buffer, 0777)
			if werr != nil {
				return werr
			}
			buffer = nil
		}
	}
	return nil
}

//SumPayments method sums up the payments using goroutines and returns
func (s *Service) SumPayments(goroutines int) types.Money {

	if goroutines < 1 {
		goroutines = 1
	}

	paysPerRoutine := (len(s.payments) / goroutines) + 1

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	sum := types.Money(0)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		partialSum := types.Money(0)

		go func(iteration int) {
			defer wg.Done()
			lowerEnd := iteration * paysPerRoutine
			higherEnd := (iteration * paysPerRoutine) + paysPerRoutine
			for j := lowerEnd; j < higherEnd; j++ {
				if j > len(s.payments)-1 {
					break
				} //break if out of range
				partialSum += s.payments[j].Amount
			}
			mu.Lock()
			defer mu.Unlock()
			sum += partialSum
		}(i)
	}
	wg.Wait()
	return sum
}

//FilterPayments method returns the slice of payments from {accountID}, using {goroutines} number of threads
func (s *Service) FilterPayments(accountID int64, goroutines int) ([]types.Payment, error) {
	_, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	if goroutines < 1 {
		goroutines = 1
	}

	paysPerRoutine := (len(s.payments) / goroutines) + 1

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	payments := make([]types.Payment, 0)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		partialPayments := make([]types.Payment, 0)

		go func(iteration int) {
			defer wg.Done()
			lowerEnd := iteration * paysPerRoutine
			higherEnd := (iteration * paysPerRoutine) + paysPerRoutine
			for j := lowerEnd; j < higherEnd; j++ {
				if j > len(s.payments)-1 {
					break
				} //break if out of range
				if s.payments[j].AccountID == accountID {
					partialPayments = append(partialPayments, *s.payments[j])
				}
			}
			mu.Lock()
			defer mu.Unlock()
			payments = append(payments, partialPayments...)
		}(i)
	}
	wg.Wait()
	return payments, nil
}

//FilterPaymentsByFn method filters payments by passed function using goroutines
func (s *Service) FilterPaymentsByFn(filter func(payment types.Payment) bool, goroutines int) ([]types.Payment, error) {

	if goroutines < 1 {
		goroutines = 1
	}

	paysPerRoutine := (len(s.payments) / goroutines) + 1

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	payments := make([]types.Payment, 0)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		partialPayments := make([]types.Payment, 0)

		go func(iteration int) {
			defer wg.Done()
			lowerEnd := iteration * paysPerRoutine
			higherEnd := (iteration * paysPerRoutine) + paysPerRoutine
			for j := lowerEnd; j < higherEnd; j++ {
				if j > len(s.payments)-1 {
					break
				} //break if out of range
				if filter(*s.payments[j]) {
					partialPayments = append(partialPayments, *s.payments[j])
				}
			}
			mu.Lock()
			defer mu.Unlock()
			payments = append(payments, partialPayments...)
		}(i)
	}
	wg.Wait()
	return payments, nil
}

//FilterMobile checks if payment's category is "mobile"
func FilterMobile(payment types.Payment) bool {
	return payment.Category == "mobile"
}

//Progress type holds the information about partial sums of big batches of payments. It's being used only in SumPaymentsByProgress method
type Progress struct {
	Part   int
	Result types.Money
}

//SumPaymentsWithProgress method utilizes channels transfering data between functions to calculate the partial sums of big equal chunks of payments
func (s *Service) SumPaymentsWithProgress() <-chan Progress {
	batchSize := 100_000
	routines := 1 + len(s.payments) / batchSize

	wg := sync.WaitGroup{}
	progressChannel := make(chan Progress, routines)
	defer close(progressChannel)

	for i := 0; i < routines; i++ {
		wg.Add(1)
		batchStart := i * batchSize
		batchEnd := (1 + i) * batchSize
		if batchEnd > len(s.payments) {
			batchEnd = len(s.payments)
		}
		subtotal := make(chan types.Money)
		go func(sub chan<- types.Money, payments []*types.Payment) {
			defer wg.Done()
			sum := types.Money(0)
			for _, pay := range payments {
				sum += pay.Amount
			}
			sub <- sum
		}(subtotal, s.payments[batchStart:batchEnd])
		progressChannel <- Progress{Part: i, Result: <-subtotal}
	}
	wg.Wait()
	return progressChannel
}