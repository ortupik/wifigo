package queue

const (
	TypeMikrotikCommand  = "mikrotik:command"
	TypeDatabaseOperation = "database:operation"
	ActionSaveMpesaCallback = "action:save_payment_callback"
	
	QueueCritical  = "critical" // For login/logout, authentication, critical DB updates
	QueueDefault   = "default"  // For regular commands, standard DB operations
	QueueReporting = "reporting" // For logs, stats collection, non-critical DB reads/writes
)