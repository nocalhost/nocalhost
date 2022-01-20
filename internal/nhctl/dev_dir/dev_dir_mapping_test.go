package dev_dir

import (
	"errors"
	"fmt"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/pkg/nhctl/log"
	"os"
	"testing"
)

var fakeKubeconfig = ""

func TestAssociate(t *testing.T) {
	os.Setenv("TEST", "-")
	defer ClearAllData()

	path := DevPath("testPath1")

	pack := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "container1",
	}

	if err := path.Associate(
		pack, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	nowAllPacks := path.GetAllPacks()

	if len(nowAllPacks.Packs) != 1 {
		t.Fatal(errors.New(fmt.Sprintf("Associate fail! %v", nowAllPacks)))
	}

	if defaultPack, err := path.GetDefaultPack(); err != nil {
		return
	} else {
		if defaultPack.Key() != pack.Key() {
			t.Fatal(errors.New("Associate fail! "))
		}
	}

	associatePath := pack.GetAssociatePath()

	if associatePath != path {
		log.Fatalf("Get Associate path with wrong value %s", associatePath)
	}
}

func TestMultipleAssociate(t *testing.T) {
	os.Setenv("TEST", "-")
	defer ClearAllData()

	path := DevPath("testPath1")

	pack := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "container1",
	}

	if err := path.Associate(
		pack, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	pack2 := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "container2",
	}

	if err := path.Associate(
		pack2, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	nowAllPacks := path.GetAllPacks()

	if len(nowAllPacks.Packs) != 2 {
		t.Fatal(errors.New(fmt.Sprintf("Associate fail! %v", nowAllPacks)))
	}

	if defaultPack, err := path.GetDefaultPack(); err != nil {
		return
	} else {
		if defaultPack.Key() != pack2.Key() {
			t.Fatal(errors.New("Associate fail! "))
		}
	}

	associatePath := pack.GetAssociatePath()

	if associatePath != path {
		log.Fatalf("Get Associate path with wrong value %s", associatePath)
	}
}

func TestMultipleUnAssociate(t *testing.T) {
	os.Setenv("TEST", "-")
	defer ClearAllData()

	path := DevPath("testPath1")

	pack := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "container1",
	}

	if err := path.Associate(
		pack, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	pack2 := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "container2",
	}

	if err := path.Associate(
		pack2, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	nowAllPacks := path.GetAllPacks()

	if len(nowAllPacks.Packs) != 2 {
		t.Fatal(errors.New(fmt.Sprintf("Associate fail! %v", nowAllPacks)))
	}

	if defaultPack, err := path.GetDefaultPack(); err != nil {
		return
	} else {
		if defaultPack.Key() != pack2.Key() {
			t.Fatal(errors.New("Associate fail! "))
		}
	}

	if err := path.RemovePack(pack2); err != nil {
		t.Fatal(err)
	}

	if defaultPack, err := path.GetDefaultPack(); err != nil {
		return
	} else {
		if defaultPack.Key() != pack.Key() {
			t.Fatal(errors.New("Associate fail! "))
		}
	}

	associatePath := pack.GetAssociatePath()

	if associatePath != path {
		log.Fatalf("Get Associate path with wrong value %s", associatePath)
	}
}

func TestGetAssociatePath(t *testing.T) {
	os.Setenv("TEST", "-")
	defer ClearAllData()

	path := DevPath("testPath1")

	pack := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "container1",
	}

	if err := path.Associate(
		pack, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	associatePath := pack.GetAssociatePath()

	if associatePath != path {
		log.Fatalf("Get Associate path with wrong value %s", associatePath)
	}
}

func TestGetAssociatePathWithNoContainer(t *testing.T) {
	os.Setenv("TEST", "-")
	defer ClearAllData()

	path := DevPath("testPath1")

	pack := &SvcPack{
		Ns:      "nocalhost",
		App:     "nh",
		SvcType: base.StatefulSet,
		Svc:     "mariadb",
	}

	if err := path.Associate(
		pack, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	anotherAssociatePack := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "whatever",
	}

	associatePath := anotherAssociatePack.GetAssociatePath()

	if associatePath != path {
		log.Fatalf("Get Associate path with wrong value %s", associatePath)
	}
}

func TestGetAssociatePathWithNoContainerAndUnAssociate(t *testing.T) {
	os.Setenv("TEST", "-")
	defer ClearAllData()

	path := DevPath("testPath1")

	pack := &SvcPack{
		Ns:      "nocalhost",
		App:     "nh",
		SvcType: base.StatefulSet,
		Svc:     "mariadb",
	}

	if err := path.Associate(
		pack, fakeKubeconfig, true,
	); err != nil {
		t.Fatal(err)
	}

	anotherAssociatePack := &SvcPack{
		Ns:        "nocalhost",
		App:       "nh",
		SvcType:   base.StatefulSet,
		Svc:       "mariadb",
		Container: "whatever",
	}

	associatePath := anotherAssociatePack.GetAssociatePath()

	if associatePath != path {
		log.Fatalf("Get Associate path with wrong value %s", associatePath)
	}

	anotherAssociatePack.UnAssociatePath()

	mustEmpty := anotherAssociatePack.GetAssociatePath()
	if mustEmpty != "" {
		log.Fatalf("Un Associate fail %s", associatePath)
	}
}
