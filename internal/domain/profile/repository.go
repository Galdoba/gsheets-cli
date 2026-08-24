package profile

// Repository defines the contract for persisting and retrieving profiles.
type Repository interface {
	// List returns all available profiles.
	List() ([]Profile, error)

	// Get retrieves a single profile by its name.
	Get(name string) (*Profile, error)

	// Save creates or updates a profile.
	Save(p *Profile) error

	// Delete removes a profile by its name.
	Delete(name string) error
}
