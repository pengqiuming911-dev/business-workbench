package app

import "math"

// rebateDecisionInput aggregates the values needed to decide whether a pending
// rebate row should be shown, whether it is currently returnable, and whether
// the page-calculated outstanding values match the detail sheet.
type rebateDecisionInput struct {
	OrderID                                      string
	SubRatio, MgmtRatio, PerfRatio               float64
	SubReturnable, MgmtReturnable                bool
	OutstandingSub, OutstandingMgmt, OutstandingPerf float64
	DetailSub, DetailMgmt, DetailPerf            *float64
	RebateTarget                                 string
}

type rebateDecision struct {
	Exclude      bool
	IsReturnable string
	CheckSub     string
	CheckMgmt    string
	CheckPerf    string
}

func computeRebateDecision(p rebateDecisionInput) rebateDecision {
	hasAnyRatio := p.SubRatio != 0 || p.MgmtRatio != 0 || p.PerfRatio != 0
	if !hasAnyRatio {
		return rebateDecision{Exclude: true}
	}

	returnable := p.SubReturnable || p.MgmtReturnable
	if returnable && p.OutstandingSub <= 0 && p.OutstandingMgmt <= 0 && p.OutstandingPerf <= 0 {
		return rebateDecision{Exclude: true}
	}

	isReturnable := "暂不可返"
	if returnable {
		isReturnable = "待返"
	}

	hasRebateTarget := p.RebateTarget != ""
	return rebateDecision{
		IsReturnable: isReturnable,
		CheckSub:     checkTF(p.OutstandingSub, p.DetailSub, hasRebateTarget),
		CheckMgmt:    checkTF(p.OutstandingMgmt, p.DetailMgmt, hasRebateTarget),
		CheckPerf:    checkTF(p.OutstandingPerf, p.DetailPerf, hasRebateTarget),
	}
}

func checkTF(pageOutstanding float64, detail *float64, hasRebateTarget bool) string {
	if detail == nil {
		if hasRebateTarget {
			return "-"
		}
		return "--"
	}
	if math.Abs(pageOutstanding-*detail) < 0.01 {
		return "T"
	}
	return "F"
}
