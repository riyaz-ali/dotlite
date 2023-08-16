package x

// #include "sql3parse_table.h"
import "C"

// This file contains golang constants for enum values defined in sql3parse_table.h
//go:generate stringer -linecomment -type OnConflict,Ordering,ForeignKeyAction,ForeignKeyDeferrable,ConstraintType -output enums_string.go

// ErrorCode represents error values that maybe returned from sql3parse routines
type ErrorCode C.sql3error_code

const (
	ErrNone        = ErrorCode(C.SQL3ERROR_NONE) // not error technically
	ErrMemory      = ErrorCode(C.SQL3ERROR_MEMORY)
	ErrSyntax      = ErrorCode(C.SQL3ERROR_SYNTAX)
	ErrUnsupported = ErrorCode(C.SQL3ERROR_UNSUPPORTEDSQL)
)

// OnConflict represents an enumeration of all possible "ON CONFLICT" clause values
type OnConflict C.sql3conflict_clause

const (
	ConflictNone     = OnConflict(C.SQL3CONFLICT_NONE)     // NOTHING
	ConflictRollback = OnConflict(C.SQL3CONFLICT_ROLLBACK) // ROLLBACK
	ConflictAbort    = OnConflict(C.SQL3CONFLICT_ABORT)    // ABORT
	ConflictFail     = OnConflict(C.SQL3CONFLICT_FAIL)     // FAIL
	ConflictIgnore   = OnConflict(C.SQL3CONFLICT_IGNORE)   // IGNORE
	ConflictReplace  = OnConflict(C.SQL3CONFLICT_REPLACE)  // REPLACE
)

// Ordering represents ordering used by the "ORDER BY" clause
type Ordering C.sql3order_clause

const (
	OrderingNone = Ordering(C.SQL3ORDER_NONE) // ORDER_NONE
	OrderingAsc  = Ordering(C.SQL3ORDER_ASC)  // ORDER_ASC
	OrderingDesc = Ordering(C.SQL3ORDER_DESC) // ORDER_DESC
)

// ForeignKeyAction represents action defined when a foreign key is updated
type ForeignKeyAction C.sql3fk_action

const (
	ForeignKeyActionNone       = ForeignKeyAction(C.SQL3FKACTION_NONE)       // SET_NONE
	ForeignKeyActionSetNull    = ForeignKeyAction(C.SQL3FKACTION_SETNULL)    // SET_NULL
	ForeignKeyActionSetDefault = ForeignKeyAction(C.SQL3FKACTION_SETDEFAULT) // SET_DEFAULT
	ForeignKeyActionCascade    = ForeignKeyAction(C.SQL3FKACTION_CASCADE)    // CASCADE
	ForeignKeyActionRestrict   = ForeignKeyAction(C.SQL3FKACTION_RESTRICT)   // RESTRICT
	ForeignKeyActionNoAction   = ForeignKeyAction(C.SQL3FKACTION_NOACTION)   // NO_ACTION
)

// ForeignKeyDeferrable represents the deferrable status of a foreign key constraint when in transaction
type ForeignKeyDeferrable C.sql3fk_deftype

const (
	ForeignKeyDeferrableNone                            = ForeignKeyDeferrable(C.SQL3DEFTYPE_NONE)                              // NONE
	ForeignKeyDeferrableDeferrable                      = ForeignKeyDeferrable(C.SQL3DEFTYPE_DEFERRABLE)                        // DEFERRABLE
	ForeignKeyDeferrableInitiallyDeferred               = ForeignKeyDeferrable(C.SQL3DEFTYPE_DEFERRABLE_INITIALLY_DEFERRED)     // DEFERRABLE_INITIALLY_DEFERRED
	ForeignKeyDeferrableInitiallyImmediate              = ForeignKeyDeferrable(C.SQL3DEFTYPE_DEFERRABLE_INITIALLY_IMMEDIATE)    // DEFERRABLE_INITIALLY_IMMEDIATE
	ForeignKeyDeferrableNotDeferrable                   = ForeignKeyDeferrable(C.SQL3DEFTYPE_NOTDEFERRABLE)                     // NOT_DEFERRABLE
	ForeignKeyDeferrableNotDeferrableInitiallyDeferred  = ForeignKeyDeferrable(C.SQL3DEFTYPE_NOTDEFERRABLE_INITIALLY_DEFERRED)  // NOT_DEFERRABLE_INITIALLY_DEFERRED
	ForeignKeyDeferrableNotDeferrableInitiallyImmediate = ForeignKeyDeferrable(C.SQL3DEFTYPE_NOTDEFERRABLE_INITIALLY_IMMEDIATE) // NOT_DEFERRABLE_INITIALLY_IMMEDIATE
)

// ConstraintType represents the type of the constraint
type ConstraintType C.sql3constraint_type

const (
	ConstraintPrimaryKey = ConstraintType(C.SQL3TABLECONSTRAINT_PRIMARYKEY) // PRIMARY_KEY
	ConstraintUnique     = ConstraintType(C.SQL3TABLECONSTRAINT_UNIQUE)     // UNIQUE
	ConstraintCheck      = ConstraintType(C.SQL3TABLECONSTRAINT_CHECK)      // CHECK
	ConstraintForeignKey = ConstraintType(C.SQL3TABLECONSTRAINT_FOREIGNKEY) // FOREIGN_KEY
)
