package model 

import "time"

// RadCheck maps to the 'radcheck' table in FreeRADIUS.
// It stores user authentication details.
type RadCheck struct {
	ID        int    `gorm:"primaryKey;autoIncrement;column:id"`
	Username  string `gorm:"column:username"` // RADIUS username (e.g., hotspot username, home user identifier)
	Attribute string `gorm:"column:attribute"`           // RADIUS attribute (e.g., "Cleartext-Password", "Expiration")
	Op        string `gorm:"column:op;default:':='"`     // Operator (e.g., ":=", "==")
	Value     string `gorm:"column:value"`               // Attribute value (e.g., user's password, expiration date)
}

// TableName overrides the table name to `radcheck`.
func (RadCheck) TableName() string {
	return "radcheck"
}

// RadReply maps to the 'radreply' table in FreeRADIUS.
// It stores attributes to be returned to the NAS (MikroTik).
type RadReply struct {
	ID        int    `gorm:"primaryKey;autoIncrement;column:id"`
	Username  string `gorm:"column:username;index:username"` // RADIUS username
	Attribute string `gorm:"column:attribute"`               // RADIUS attribute (e.g., "Session-Timeout", "Acct-Interim-Interval", "Mikrotik-Rate-Limit")
	Op        string `gorm:"column:op;default:':='"`         // Operator (e.g., ":=", "=")
	Value     string `gorm:"column:value"`                   // Attribute value (e.g., "3600", "600", "1M/1M")
}

// TableName overrides the table name to `radreply`.
func (RadReply) TableName() string {
	return "radreply"
}

// RadUserGroup maps to the 'radusergroup' table in FreeRADIUS.
// It assigns users to groups for applying group-based attributes.
type RadUserGroup struct {
	ID        int    `gorm:"primaryKey;autoIncrement;column:id"`
	Username  string `gorm:"column:username;index:username"`   // RADIUS username
	Groupname string `gorm:"column:groupname;index:groupname"` // Name of the group (e.g., "hotspot_premium", "home_standard")
	Priority  int    `gorm:"column:priority;default:1"`        // Priority for applying attributes (lower is higher priority)
}

// TableName overrides the table name to `radusergroup`.
func (RadUserGroup) TableName() string {
	return "radusergroup"
}

// RadGroupCheck maps to the 'radgroupcheck' table in FreeRADIUS.
// It stores authentication attributes for groups.
type RadGroupCheck struct {
	ID        int    `gorm:"primaryKey;autoIncrement;column:id"`
	Groupname string `gorm:"column:groupname;index:groupname"` // Name of the group
	Attribute string `gorm:"column:attribute"`                 // RADIUS attribute
	Op        string `gorm:"column:op;default:':='""`           // Operator
	Value     string `gorm:"column:value"`                     // Attribute value
}

// TableName overrides the table name to `radgroupcheck`.
func (RadGroupCheck) TableName() string {
	return "radgroupcheck"
}

// RadGroupReply maps to the 'radgroupreply' table in FreeRADIUS.
// It stores reply attributes for groups.
type RadGroupReply struct {
	ID        int    `gorm:"primaryKey;autoIncrement;column:id"`
	Groupname string `gorm:"column:groupname;index:groupname"` // Name of the group
	Attribute string `gorm:"column:attribute"`                 // RADIUS attribute
	Op        string `gorm:"column:op;default:':='""`           // Operator
	Value     string `gorm:"column:value"`                     // Attribute value
}

// TableName overrides the table name to `radgroupreply`.
func (RadGroupReply) TableName() string {
	return "radgroupreply"
}

// RadAcct maps to the 'radacct' table in FreeRADIUS.
// It stores session accounting information.
type RadAcct struct {
	RadAcctID          int64      `gorm:"primaryKey;autoIncrement;column:radacctid"`
	AcctSessionID      string     `gorm:"uniqueIndex;column:acctsessionid"`    // Unique session ID
	AcctUniqueID       string     `gorm:"uniqueIndex;column:acctuniqueid"`     // Unique accounting request ID
	Username           string     `gorm:"column:username;index:username"`
	Groupname          string     `gorm:"column:groupname"`
	Realm              *string    `gorm:"column:realm"`                          // Nullable
	NasIPAddress       string     `gorm:"column:nasipaddress;index:nasipaddress"`// IP address of the Network Access Server (MikroTik)
	NasPortID          *string    `gorm:"column:nasportid"`                      // Nullable
	NasPortType        *string    `gorm:"column:nasporttype"`                    // Nullable
	// Removed type:timestamp from nullable time fields
	AcctStartTime      *time.Time `gorm:"column:acctstarttime;index:acctstarttime"`
	AcctUpdateTime     *time.Time `gorm:"column:acctupdatetime"`
	AcctStopTime       *time.Time `gorm:"column:acctstoptime"`
	AcctSessionTime    *int       `gorm:"column:acctsessiontime;index:acctsessiontime"`// Nullable
	AcctInputOctets    *int64     `gorm:"column:acctinputoctets;index:acctinputoctets"`// Nullable
	AcctOutputOctets   *int64     `gorm:"column:acctoutputoctets;index:acctoutputoctets"`// Nullable
	CalledStationID    string     `gorm:"column:calledstationid;index:calledstationid"`// NAS identifier (e.g., MikroTik router name)
	CallingStationID   string     `gorm:"column:callingstationid;index:callingstationid"`// Client identifier (e.g., MAC address)
	AcctTerminateCause *string    `gorm:"column:acctterminatecause"`             // Nullable
	ServiceType        *string    `gorm:"column:servicetype"`                    // Nullable
	FramedProtocol     *string    `gorm:"column:framedprotocol"`                 // Nullable
	FramedIPAddress    *string    `gorm:"column:framedipaddress"`                // Nullable
	ConnectInfoStart   *string    `gorm:"column:connectinfo_start"`              // Nullable
	ConnectInfoStop    *string    `gorm:"column:connectinfo_stop"`               // Nullable
	EventTimestamp     *time.Time `gorm:"column:eventtimestamp"`
}

// TableName overrides the table name to `radacct`.
func (RadAcct) TableName() string {
	return "radacct"
}