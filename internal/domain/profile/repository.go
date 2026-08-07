package profile

// Repository defines the contract for persisting and retrieving profiles.
type Repository interface {
	// List returns all profiles for a specific sheet and table.
	List(sheetID, tableName string) ([]Profile, error)

	// Get retrieves a single profile by its ID.
	Get(id string) (*Profile, error)

	// Save creates or updates a profile.
	Save(p *Profile) error

	// Delete removes a profile by its ID.
	Delete(id string) error
}
