package pgdoctor

func DemoSamples() []Sample {
	return []Sample{
		{QueryID: 8412, Fingerprint: "select * from cdr where queue = $1", Table: "cdr", Calls: 1204, MeanMs: 940, TotalMs: 1131760, Rows: 2400000, SeqScans: 1180, IdxScans: 24, Plan: "started_at::queue"},
		{QueryID: 2211, Fingerprint: "update agents set state = $1 where ext = $2", Table: "agents", Calls: 88420, MeanMs: 1.9, TotalMs: 168000, Rows: 88420, SeqScans: 0, IdxScans: 88420, Plan: "ext"},
		{QueryID: 5567, Fingerprint: "select count(*) from cdr where sla_met = false", Table: "cdr", Calls: 312, MeanMs: 420, TotalMs: 131040, Rows: 910000, SeqScans: 300, IdxScans: 12, Plan: "sla_met::queue"},
		{QueryID: 9930, Fingerprint: "insert into alerts (...) values (...)", Table: "alerts", Calls: 51200, MeanMs: 0.8, TotalMs: 40960, Rows: 51200, SeqScans: 0, IdxScans: 0, Plan: "created_at"},
		{QueryID: 1187, Fingerprint: "select * from query_samples where fingerprint = $1 order by captured_at desc", Table: "query_samples", Calls: 2400, MeanMs: 310, TotalMs: 744000, Rows: 1800000, SeqScans: 2390, IdxScans: 10, Plan: "fingerprint::captured_at"},
	}
}

func DemoDeadlocks() []Deadlock {
	return []Deadlock{
		{
			At:       "2026-11-08 09:14:02 MSK",
			Victim:   "process 18422",
			WaitsFor: "ShareLock on transaction 9912",
			LockType: "ShareLock",
			Relation: "telephony.agents",
			Advice:   "упорядочить захват блокировок: сначала agents, затем queues",
		},
		{
			At:       "2026-11-08 11:02:41 MSK",
			Victim:   "process 20114",
			WaitsFor: "transactionid 10423",
			LockType: "tuple lock",
			Relation: "telephony.cdr_2026_11",
			Advice:   "сократить транзакцию и вынести вычисления до UPDATE",
		},
	}
}

func DemoLog() string {
	return `2026-11-08 09:14:02 MSK ERROR:  deadlock detected
DETAIL:  Process 18422 waits for ShareLock on transaction 9912; blocked by process 18401.
	Process 18401 waits for ShareLock on transaction 9908; blocked by process 18422.
	Process 18422: update agents set state = 'busy' where ext = '101'
	Process 18401: update agents set state = 'ready' where ext = '104'`
}
