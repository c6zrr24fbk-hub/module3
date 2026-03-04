package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE parcel (
			number INTEGER PRIMARY KEY AUTOINCREMENT,
			client INTEGER NOT NULL,
			status TEXT NOT NULL,
			address TEXT NOT NULL,
			created_at TEXT NOT NULL
		)
	`)
	require.NoError(t, err)
	return db
}

func TestAddGetDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewParcelStore(db)

	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)

	p, err := store.Get(id)
	require.NoError(t, err)

	expected := parcel
	expected.Number = id
	require.Equal(t, expected, p)

	err = store.Delete(id)
	require.NoError(t, err)

	_, err = store.Get(id)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteNonRegistered(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewParcelStore(db)

	parcel := getTestParcel()
	parcel.Status = ParcelStatusSent
	id, err := store.Add(parcel)
	require.NoError(t, err)

	err = store.Delete(id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot delete parcel")

	p, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, parcel.Status, p.Status) // статус не изменился
}

func TestSetAddress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewParcelStore(db)

	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)

	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err)

	p, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, newAddress, p.Address)

	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	err = store.SetAddress(id, "another address")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot change address")

	p, err = store.Get(id)
	require.NoError(t, err)
	require.Equal(t, newAddress, p.Address) // адрес не изменился
}

func TestSetStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewParcelStore(db)

	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)

	newStatus := ParcelStatusSent
	err = store.SetStatus(id, newStatus)
	require.NoError(t, err)

	p, err := store.Get(id)
	require.NoError(t, err)

	expected := parcel
	expected.Number = id
	expected.Status = newStatus
	require.Equal(t, expected, p)
}

func TestGetByClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewParcelStore(db)

	client := 12345
	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	statuses := []string{ParcelStatusRegistered, ParcelStatusSent, ParcelStatusDelivered}
	for i := range parcels {
		parcels[i].Client = client
		parcels[i].Status = statuses[i]
		parcels[i].Address = fmt.Sprintf("addr%d", i+1)
	}
	parcelMap := make(map[int]Parcel)

	for i, p := range parcels {
		id, err := store.Add(p)
		require.NoError(t, err)
		parcels[i].Number = id
		parcelMap[id] = parcels[i]
	}

	stored, err := store.GetByClient(client)
	require.NoError(t, err)
	require.Len(t, stored, len(parcels))

	for _, sp := range stored {
		original, ok := parcelMap[sp.Number]
		require.True(t, ok, "unexpected parcel number %d", sp.Number)
		require.Equal(t, original, sp)
	}
}
