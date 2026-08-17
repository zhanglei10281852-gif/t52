package ticket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"scenic-ticket/config"
	"scenic-ticket/models"
	"time"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type TicketNumberGenerator interface {
	Generate(visitDate time.Time) (string, error)
}

type RandomTicketNumberGenerator struct{}

func (RandomTicketNumberGenerator) Generate(visitDate time.Time) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return visitDate.Format("20060102") + hex.EncodeToString(random), nil
}

type Service struct {
	repository Repository
	clock      Clock
	numbers    TicketNumberGenerator
}

func NewService(repository Repository, clock Clock, numbers TicketNumberGenerator) *Service {
	return &Service{repository: repository, clock: clock, numbers: numbers}
}

type SellInput struct {
	TicketTypeID uint
	BuyerName    string
	Phone        string
	VisitDate    time.Time
	TimeSlotID   int
	SellerID     uint
	SellerName   string
}

type SellResult struct {
	TicketNo string
	Message  string
}

func (s *Service) Sell(ctx context.Context, input SellInput) (SellResult, error) {
	if input.TicketTypeID == 0 || input.BuyerName == "" || input.Phone == "" || input.TimeSlotID == 0 || input.SellerID == 0 {
		return SellResult{}, newError(ErrValidation, "参数错误", nil)
	}
	slotInfo := config.GetTimeSlotInfo(input.TimeSlotID)
	if slotInfo == nil {
		return SellResult{}, newError(ErrValidation, "无效的时段", nil)
	}
	visitDate := config.DateKey(input.VisitDate)
	ticketNo, err := s.numbers.Generate(visitDate)
	if err != nil {
		return SellResult{}, newError(ErrStorage, "生成票号失败", err)
	}

	err = s.repository.WithTx(ctx, func(tx TxRepository) error {
		ticketType, err := tx.FindTicketType(ctx, input.TicketTypeID)
		if errors.Is(err, ErrRecordNotFound) {
			return newError(ErrValidation, "票型不存在", nil)
		}
		if err != nil {
			return storageError("获取票型失败", err)
		}

		quota, err := tx.GetOrCreatePurchaseQuota(ctx, visitDate, input.Phone)
		if err != nil {
			return storageError("获取限购信息失败", err)
		}
		if quota.ActiveCount >= config.MaxTicketsPerPhone {
			return newError(ErrConflict, fmt.Sprintf("同一手机号每天限购%d张票", config.MaxTicketsPerPhone), nil)
		}

		dailyCapacity, err := tx.GetOrCreateDailyCapacity(ctx, visitDate)
		if err != nil {
			return storageError("获取日容量信息失败", err)
		}
		maxSell := int(float64(dailyCapacity.MaxCapacity) * config.CapacityOverSell)
		if dailyCapacity.SoldCount >= maxSell {
			return newError(ErrConflict, "当日票已售罄", nil)
		}

		slotCapacity, err := tx.GetOrCreateSlotCapacity(ctx, visitDate, input.TimeSlotID)
		if err != nil {
			return storageError("获取时段容量信息失败", err)
		}
		if slotCapacity.SoldCount >= slotCapacity.MaxCapacity {
			return newError(ErrConflict, "该时段票已售罄", nil)
		}

		value := models.Ticket{
			TicketNo:     ticketNo,
			TicketTypeID: input.TicketTypeID,
			BuyerName:    input.BuyerName,
			Phone:        input.Phone,
			VisitDate:    visitDate,
			TimeSlotID:   input.TimeSlotID,
			TimeSlotName: slotInfo.Name,
			Price:        ticketType.Price,
			Status:       models.TicketStatusUnused,
			SellerID:     input.SellerID,
			SellerName:   input.SellerName,
		}
		if err := tx.CreateTicket(ctx, &value); err != nil {
			return storageError("创建票务失败", err)
		}

		quota.ActiveCount++
		dailyCapacity.SoldCount++
		slotCapacity.SoldCount++
		if err := tx.SavePurchaseQuota(ctx, quota); err != nil {
			return storageError("更新限购信息失败", err)
		}
		if err := tx.SaveDailyCapacity(ctx, dailyCapacity); err != nil {
			return storageError("更新日容量失败", err)
		}
		if err := tx.SaveSlotCapacity(ctx, slotCapacity); err != nil {
			return storageError("更新时段容量失败", err)
		}
		return nil
	})
	if err != nil {
		return SellResult{}, normalizeTransactionError(err)
	}

	return SellResult{TicketNo: ticketNo, Message: "售票成功"}, nil
}

func (s *Service) Refund(ctx context.Context, ticketNo string) error {
	if ticketNo == "" {
		return newError(ErrValidation, "参数错误", nil)
	}
	err := s.repository.WithTx(ctx, func(tx TxRepository) error {
		value, err := tx.FindTicketForUpdate(ctx, ticketNo)
		if errors.Is(err, ErrRecordNotFound) {
			return newError(ErrNotFound, "票不存在", nil)
		}
		if err != nil {
			return storageError("获取票务失败", err)
		}
		if value.Status == models.TicketStatusRefunded {
			return newError(ErrConflict, "该票已退票", nil)
		}
		if value.Status == models.TicketStatusUsed {
			return newError(ErrConflict, "已使用的票不能退票", nil)
		}

		quota, err := tx.GetOrCreatePurchaseQuota(ctx, value.VisitDate, value.Phone)
		if err != nil {
			return storageError("获取限购信息失败", err)
		}
		dailyCapacity, err := tx.GetOrCreateDailyCapacity(ctx, value.VisitDate)
		if err != nil {
			return storageError("获取日容量信息失败", err)
		}
		slotCapacity, err := tx.GetOrCreateSlotCapacity(ctx, value.VisitDate, value.TimeSlotID)
		if err != nil {
			return storageError("获取时段容量信息失败", err)
		}
		if quota.ActiveCount < 1 || dailyCapacity.SoldCount < 1 || slotCapacity.SoldCount < 1 {
			return newError(ErrStorage, "票务容量数据不一致", nil)
		}

		value.Status = models.TicketStatusRefunded
		quota.ActiveCount--
		dailyCapacity.SoldCount--
		slotCapacity.SoldCount--
		if err := tx.SaveTicket(ctx, value); err != nil {
			return storageError("退票失败", err)
		}
		if err := tx.SavePurchaseQuota(ctx, quota); err != nil {
			return storageError("更新限购信息失败", err)
		}
		if err := tx.SaveDailyCapacity(ctx, dailyCapacity); err != nil {
			return storageError("更新日容量失败", err)
		}
		if err := tx.SaveSlotCapacity(ctx, slotCapacity); err != nil {
			return storageError("更新时段容量失败", err)
		}
		return nil
	})
	return normalizeTransactionError(err)
}

type CheckInput struct {
	TicketNo string
	GateID   uint
}

type CheckResult struct {
	TicketNo  string
	CheckTime time.Time
	GateName  string
	Message   string
}

func (s *Service) CheckIn(ctx context.Context, input CheckInput) (CheckResult, error) {
	if input.TicketNo == "" || input.GateID == 0 {
		return CheckResult{}, newError(ErrValidation, "参数错误", nil)
	}
	now := s.clock.Now()
	today := config.DateKey(now)
	var result CheckResult
	err := s.repository.WithTx(ctx, func(tx TxRepository) error {
		value, err := tx.FindTicketForUpdate(ctx, input.TicketNo)
		if errors.Is(err, ErrRecordNotFound) {
			return newError(ErrNotFound, "票不存在", nil)
		}
		if err != nil {
			return storageError("获取票务失败", err)
		}
		if value.Status == models.TicketStatusRefunded {
			return newError(ErrConflict, "该票已退票", nil)
		}
		if value.Status == models.TicketStatusUsed {
			return newError(ErrConflict, "该票已使用", nil)
		}
		if !config.DateKey(value.VisitDate).Equal(today) {
			return newError(ErrConflict, "该票不是今天的票", nil)
		}

		if slotInfo := config.GetTimeSlotInfo(value.TimeSlotID); slotInfo != nil {
			dayStart, _ := config.DayRange(today)
			slotStart := dayStart.Add(time.Duration(slotInfo.StartHour) * time.Hour)
			if now.Before(slotStart.Add(-30 * time.Minute)) {
				return newError(ErrConflict, "未到入园时间，请提前30分钟内入园", nil)
			}
		}

		gate, err := tx.FindGate(ctx, input.GateID)
		if errors.Is(err, ErrRecordNotFound) {
			return newError(ErrValidation, "闸机不存在", nil)
		}
		if err != nil {
			return storageError("获取闸机失败", err)
		}
		dailyCapacity, err := tx.GetOrCreateDailyCapacity(ctx, today)
		if err != nil {
			return storageError("获取日容量信息失败", err)
		}
		checkedOut, err := tx.CountCheckedOut(ctx, today)
		if err != nil {
			return storageError("获取在园人数失败", err)
		}
		if dailyCapacity.CheckedInCount-int(checkedOut) >= dailyCapacity.MaxCapacity {
			return newError(ErrConflict, "景区满员", nil)
		}
		slotCapacity, err := tx.GetOrCreateSlotCapacity(ctx, today, value.TimeSlotID)
		if err != nil {
			return storageError("获取时段容量信息失败", err)
		}

		gateID := gate.ID
		value.Status = models.TicketStatusUsed
		value.CheckInGateID = &gateID
		value.CheckInTime = &now
		record := models.CheckRecord{
			TicketID: value.ID, TicketNo: value.TicketNo, GateID: gate.ID, GateName: gate.Name,
			CheckType: models.CheckTypeIn, CheckTime: now, TimeSlotID: value.TimeSlotID,
		}
		dailyCapacity.CheckedInCount++
		slotCapacity.CheckedInCount++
		if err := tx.SaveTicket(ctx, value); err != nil {
			return storageError("更新票状态失败", err)
		}
		if err := tx.CreateCheckRecord(ctx, &record); err != nil {
			return storageError("创建核销记录失败", err)
		}
		if err := tx.SaveDailyCapacity(ctx, dailyCapacity); err != nil {
			return storageError("更新日容量失败", err)
		}
		if err := tx.SaveSlotCapacity(ctx, slotCapacity); err != nil {
			return storageError("更新时段容量失败", err)
		}
		result = CheckResult{TicketNo: value.TicketNo, CheckTime: now, GateName: gate.Name, Message: "入园成功"}
		return nil
	})
	if err != nil {
		return CheckResult{}, normalizeTransactionError(err)
	}
	return result, nil
}

func (s *Service) CheckOut(ctx context.Context, input CheckInput) (CheckResult, error) {
	if input.TicketNo == "" || input.GateID == 0 {
		return CheckResult{}, newError(ErrValidation, "参数错误", nil)
	}
	now := s.clock.Now()
	var result CheckResult
	err := s.repository.WithTx(ctx, func(tx TxRepository) error {
		value, err := tx.FindTicketForUpdate(ctx, input.TicketNo)
		if errors.Is(err, ErrRecordNotFound) {
			return newError(ErrNotFound, "票不存在", nil)
		}
		if err != nil {
			return storageError("获取票务失败", err)
		}
		if value.Status != models.TicketStatusUsed {
			return newError(ErrConflict, "该票未入园", nil)
		}
		if value.CheckOutTime != nil {
			return newError(ErrConflict, "该票已出园", nil)
		}
		gate, err := tx.FindGate(ctx, input.GateID)
		if errors.Is(err, ErrRecordNotFound) {
			return newError(ErrValidation, "闸机不存在", nil)
		}
		if err != nil {
			return storageError("获取闸机失败", err)
		}

		gateID := gate.ID
		value.CheckOutGateID = &gateID
		value.CheckOutTime = &now
		record := models.CheckRecord{
			TicketID: value.ID, TicketNo: value.TicketNo, GateID: gate.ID, GateName: gate.Name,
			CheckType: models.CheckTypeOut, CheckTime: now, TimeSlotID: value.TimeSlotID,
		}
		if err := tx.SaveTicket(ctx, value); err != nil {
			return storageError("更新票状态失败", err)
		}
		if err := tx.CreateCheckRecord(ctx, &record); err != nil {
			return storageError("创建出园记录失败", err)
		}
		result = CheckResult{TicketNo: value.TicketNo, CheckTime: now, GateName: gate.Name, Message: "出园成功"}
		return nil
	})
	if err != nil {
		return CheckResult{}, normalizeTransactionError(err)
	}
	return result, nil
}

func storageError(message string, cause error) error {
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return cause
	}
	return newError(ErrStorage, message, cause)
}

func normalizeTransactionError(err error) error {
	if err == nil || errors.Is(err, ErrValidation) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrConflict) || errors.Is(err, ErrStorage) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return newError(ErrStorage, "保存票务状态失败", err)
}
