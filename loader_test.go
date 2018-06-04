package main

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func setUpFs(rootDir string, vals map[string]string) error {
	for fname, val := range vals {
		p := path.Join(rootDir, fname)
		dir := path.Dir(p)
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}

		if err := ioutil.WriteFile(p, []byte(val), 0666); err != nil {
			return err
		}
	}
	return nil
}

func testDir(t *testing.T) string {
	tmpdir, err := ioutil.TempDir("", "syncret")
	if err != nil {
		t.Fatalf("erred creating test tmp dir: %v", err)
	}
	return tmpdir
}

func Test_loader_LoadAll(t *testing.T) {
	type fields struct {
		secretSuffix      string
		descriptionSuffix string
		patternSuffix     string
		decryptCmd        string
	}
	type args struct {
		fnames []string
		paths  map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []secret
		wantErr bool
	}{
		{
			"load simple",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"test_path.txt"},
				map[string]string{
					"test_path.txt": "test_value",
				},
			},
			[]secret{{Name: "/test_path", Value: "test_value"}},
			false,
		},
		{
			"finds description",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"test_path.txt"},
				map[string]string{
					"test_path.txt":         "test_value",
					"test_path.description": "a test value",
				},
			},
			[]secret{{Name: "/test_path", Value: "test_value", Description: "a test value"}},
			false,
		},
		{
			"finds pattern",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"test_path.txt"},
				map[string]string{
					"test_path.txt":     "test_value",
					"test_path.pattern": "a test pattern",
				},
			},
			[]secret{{Name: "/test_path", Value: "test_value", Pattern: "a test pattern"}},
			false,
		},
		{
			"finds all from secret",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"test_path.txt"},
				map[string]string{
					"test_path.txt":         "test_value",
					"test_path.description": "a test description",
					"test_path.pattern":     "a test pattern",
				},
			},
			[]secret{{
				Name:        "/test_path",
				Value:       "test_value",
				Description: "a test description",
				Pattern:     "a test pattern",
			}},
			false,
		},
		{
			"drops dupes",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"test_path.txt", "test_path2.txt", "test_path2.description"},
				map[string]string{
					"test_path.txt":          "test_value",
					"test_path2.txt":         "a test pattern",
					"test_path2.description": "a second test path",
				},
			},
			[]secret{
				{Name: "/test_path", Value: "test_value"},
				{Name: "/test_path2", Value: "a test pattern", Description: "a second test path"},
			},
			false,
		},
		{
			"nested is fine",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"hi/test_path.txt"},
				map[string]string{
					"hi/test_path.txt":         "test_value",
					"hi/test_path.description": "a test description",
					"hi/test_path.pattern":     "a test pattern",
				},
			},
			[]secret{{
				Name:        "/hi/test_path",
				Value:       "test_value",
				Description: "a test description",
				Pattern:     "a test pattern",
			}},
			false,
		},
		{
			"missing secret is an error",
			fields{
				secretSuffix:      ".txt",
				descriptionSuffix: ".description",
				patternSuffix:     ".pattern",
				decryptCmd:        "cat",
			},
			args{
				[]string{"hi/test_path.txt"},
				map[string]string{
					"test_path.description": "a test description",
					"test_path.pattern":     "a test pattern",
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := testDir(t)
			defer os.RemoveAll(tmpdir)

			setUpFs(tmpdir, tt.args.paths)

			l := fsLoader{
				secretSuffix:      tt.fields.secretSuffix,
				descriptionSuffix: tt.fields.descriptionSuffix,
				patternSuffix:     tt.fields.patternSuffix,
				decryptCmd:        tt.fields.decryptCmd,
				rootDir:           tmpdir,
			}
			got, err := l.LoadAll(tt.args.fnames)
			if (err != nil) != tt.wantErr {
				t.Errorf("fsLoader.LoadAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fsLoader.LoadAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readVal(t *testing.T) {
	type args struct {
		fname  string
		data   []byte
		exists bool
		mode   os.FileMode
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"nonexistent",
			args{
				"nonexistent_filename",
				nil,
				false,
				0,
			},
			nil,
			false,
		},
		{
			"present",
			args{
				"present_filename",
				[]byte("1234"),
				true,
				0666,
			},
			[]byte("1234"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := testDir(t)
			defer os.RemoveAll(tmpdir)

			p := path.Join(tmpdir, tt.args.fname)
			if tt.args.exists {
				if err := ioutil.WriteFile(p, tt.args.data, tt.args.mode); err != nil {
					t.Fatalf("Failed writing test dtata to %v: %v", p, err)
				}
			}

			got, err := readVal(p)
			if (err != nil) != tt.wantErr {
				t.Errorf("readVal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readVal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decrypt(t *testing.T) {
	type args struct {
		decryptCmd string
		path       string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"simple",
			args{
				"cat",
				"mypath",
			},
			[]byte("thisisjoe"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := testDir(t)
			defer os.RemoveAll(tmpdir)

			p := path.Join(tmpdir, tt.args.path)
			if err := ioutil.WriteFile(p, tt.want, 0666); err != nil {
				t.Fatalf("erred writing test data to %v: %v", p, err)
			}

			got, err := decrypt(tt.args.decryptCmd, p)
			if (err != nil) != tt.wantErr {
				t.Errorf("decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decrypt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unextended(t *testing.T) {
	type args struct {
		path       string
		extensions []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"one to one",
			args{
				"test_path.txt",
				[]string{
					".txt",
				},
			},
			"test_path",
		},
		{
			"none",
			args{
				"test_path",
				[]string{
					".txt",
				},
			},
			"",
		},
		{
			"unrecognized",
			args{
				"test_path.gpg",
				[]string{
					".txt",
				},
			},
			"",
		},
		{
			"many to one",
			args{
				"test_path.txt",
				[]string{
					".gpg",
					".txt",
				},
			},
			"test_path",
		},
		{
			"many to none",
			args{
				"test_path.gz",
				[]string{
					".gpg",
					".txt",
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unextended(tt.args.path, tt.args.extensions...); got != tt.want {
				t.Errorf("unextended() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newLoader(t *testing.T) {
	type args struct {
		env     map[string]string
		rootDir string
		prefix  string
		trim    bool
	}
	tests := []struct {
		name    string
		args    args
		want    fsLoader
		wantErr bool
	}{
		{
			"defaults",
			args{
				map[string]string{},
				"",
				"",
				true,
			},
			fsLoader{
				".gpg",
				".description",
				".pattern",
				"cat",
				"",
				"",
				true,
			},
			false,
		},
		{
			"overrides",
			args{
				map[string]string{
					decryptEnvVar:     "gpg",
					secretEnvVar:      ".txt",
					descriptionEnvVar: ".desc",
					patternEnvVar:     ".patt",
				},
				"/tmp",
				"blah/",
				false,
			},
			fsLoader{
				".txt",
				".desc",
				".patt",
				"gpg",
				"/tmp",
				"blah/",
				false,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newLoader(tt.args.env, tt.args.rootDir, tt.args.prefix, tt.args.trim)
			if (err != nil) != tt.wantErr {
				t.Errorf("newLoader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newLoader() = %v, want %v", got, tt.want)
			}
		})
	}
}
