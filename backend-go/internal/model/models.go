package model

type Product struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	IsMain             *int     `json:"is_main,omitempty"`
	IssueDate          string   `json:"issue_date"`
	CompleteDate       string   `json:"complete_date"`
	SubscribeAmount    *float64 `json:"subscribe_amount,omitempty"`
	OutstandingAmount  *float64 `json:"outstanding_amount,omitempty"`
	Manager            string   `json:"manager"`
	HoldingStatus      string   `json:"holding_status"`
	StructureType      string   `json:"structure_type"`
	Code               string   `json:"code"`
	LockDays           *int     `json:"lock_days,omitempty"`
	LockMonths         *int     `json:"lock_months,omitempty"`
	FirstKnockoutRatio *float64 `json:"first_knockout_ratio,omitempty"`
	EntryPrice         *float64 `json:"entry_price,omitempty"`
	MonthlyDecrease    *float64 `json:"monthly_decrease,omitempty"`
	Term               string   `json:"term"`
	Parachute          string   `json:"parachute"`
	DividendBarrier    *float64 `json:"dividend_barrier,omitempty"`
	MonthlyCoupon      *float64 `json:"monthly_coupon,omitempty"`
	Coupon1st          *float64 `json:"coupon_1st,omitempty"`
	Coupon2nd          *float64 `json:"coupon_2nd,omitempty"`
	Coupon3rd          *float64 `json:"coupon_3rd,omitempty"`
	DurationMonths     *float64 `json:"duration_months,omitempty"`
	AbsoluteReturn     *float64 `json:"absolute_return,omitempty"`
	HolidayAdjust      string   `json:"holiday_adjust"`
	Raw                string   `json:"raw,omitempty"`
	KnockIn            string   `json:"knock_in"`
	DurationDays       *int     `json:"duration_days,omitempty"`
	KnockedIn          string   `json:"knocked_in"`
	MarginRatio        *float64 `json:"margin_ratio,omitempty"`
	Custodian          string   `json:"custodian"`
	Counterparty       string   `json:"counterparty"`
}

type TransactionRow struct {
	ID                  int64    `json:"id"`
	TransactionDate     string   `json:"transaction_date"`
	FlightID            string   `json:"flight_id"`
	Counterparty        string   `json:"counterparty"`
	SubscribeAmount     *float64 `json:"subscribe_amount,omitempty"`
	ProductName         string   `json:"product_name"`
	CustomerName        string   `json:"customer_name"`
	ActualBuyer         string   `json:"actual_buyer"`
	Amount              *float64 `json:"amount,omitempty"`
	SubscribeFeeRatio   *float64 `json:"subscribe_fee_ratio,omitempty"`
	ManagementFeeRatio  *float64 `json:"management_fee_ratio,omitempty"`
	PerformanceFeeRatio *float64 `json:"performance_fee_ratio,omitempty"`
	RebateTarget        string   `json:"rebate_target"`
	FlightDate          string   `json:"flight_date"`
	HoldingStatus       string   `json:"holding_status"`
	CompleteDate        string   `json:"complete_date"`
	Underlying          string   `json:"underlying"`
	StructureType       string   `json:"structure_type"`
	LockPeriod          string   `json:"lock_period"`
	DividendBarrier     *float64 `json:"dividend_barrier,omitempty"`
	MonthlyCoupon       *float64 `json:"monthly_coupon,omitempty"`
	Coupon1st           *float64 `json:"coupon_1st,omitempty"`
	Raw                 string   `json:"raw,omitempty"`
}

type Observation struct {
	ID               int64    `json:"id"`
	ProductID        string   `json:"product_id"`
	ObservationDate  string   `json:"observation_date"`
	KnockoutPrice    *float64 `json:"knockout_price"`
	DividendLine     *float64 `json:"dividend_line"`
	UnderlyingPrice  *float64 `json:"underlying_price"`
	IsKnockedOut     string   `json:"is_knocked_out"`
	IsDividend       string   `json:"is_dividend"`
	MonthsSinceEntry *int     `json:"months_since_entry"`
	UpdatedAt        string   `json:"updated_at"`
}

type CalendarProduct struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Manager                string   `json:"manager"`
	Code                   string   `json:"code"`
	MonthsSinceEntry       int      `json:"months_since_entry"`
	EntryPrice             *float64 `json:"entry_price"`
	KnockoutPrice          *float64 `json:"knockout_price"`
	DividendLine           *float64 `json:"dividend_line"`
	IsKnockoutObservable   bool     `json:"is_knockout_observable"`
	HasDividendObservation bool     `json:"has_dividend_observation"`
}

type CalendarDay struct {
	Date     string            `json:"date"`
	Products []CalendarProduct `json:"products"`
}

type Poster struct {
	ID                   int64    `json:"id"`
	ProductID            string   `json:"product_id"`
	PosterType           string   `json:"poster_type"`
	ObservationDate      string   `json:"observation_date"`
	ProductName          string   `json:"product_name"`
	DateDisplay          string   `json:"date_display"`
	MonthsSinceEntry     *int     `json:"months_since_entry"`
	UnderlyingName       string   `json:"underlying_name"`
	AbsoluteReturn       *float64 `json:"absolute_return"`
	AnnualizedReturn     *float64 `json:"annualized_return"`
	DurationMonths       *int     `json:"duration_months"`
	ParachuteValue       string   `json:"parachute_value"`
	KnockoutValue        string   `json:"knockout_value"`
	DividendBarrierValue string   `json:"dividend_barrier_value"`
	DividendCount        *int     `json:"dividend_count"`
	CumulativeRate       *float64 `json:"cumulative_rate"`
	MonthlyCoupon        *float64 `json:"monthly_coupon"`
	EntryDate            string   `json:"entry_date"`
	CreatedAt            string   `json:"created_at"`
}

type ActivityLog struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"createdAt"`
}

type PushConfig struct {
	WebhookURL     string `json:"webhook_url"`
	CronHour       int    `json:"cron_hour"`
	CronMinute     int    `json:"cron_minute"`
	Enabled        bool   `json:"enabled"`
	LastPushTime   string `json:"last_push_time"`
	LastPushResult string `json:"last_push_result"`
}

type AgentConversation struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type AgentMessage struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	ToolCalls      string `json:"tool_calls"`
	ToolCallID     string `json:"tool_call_id"`
	CreatedAt      string `json:"created_at"`
}
