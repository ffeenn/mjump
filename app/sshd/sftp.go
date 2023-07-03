package sshd

import (
	"fmt"
	"io"
	config "mjump/app/data"
	"mjump/app/logger"
	"mjump/app/proxy"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"

	"github.com/pkg/sftp"
)

type listerat []os.FileInfo

func (l listerat) ListAt(f []os.FileInfo, offset int64) (int, error) {
	var n int
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n = copy(f, l[offset:])
	if n < len(f) {
		return n, io.EOF
	}
	return n, nil
}

type uConn struct {
	ID       string
	Addr     string
	Dirs     map[string]os.FileInfo
	fc       map[string]*fCon
	m        sync.Mutex
	modeTime time.Time
	// logChan  chan *model.FTPLog

	// closed chan struct{}
	// searchDir *SearchResultDir

	// jmsService *service.JMService
}

// func (u *uConn) Close() {
// for _, dir := range u.Dirs {
// 	if nodeDir, ok := dir.(*NodeDir); ok {
// 		nodeDir.close()
// 		continue
// 	}
// 	if assetDir, ok := dir.(*AssetDir); ok {
// 		assetDir.close()
// 		continue
// 	}
// }
// if u.searchDir != nil {
// 	u.searchDir.close()
// }
// 	close(u.closed)
// }

func NewWriterAt(f *sftp.File) io.WriterAt {
	return &clientReadWritAt{f: f, m: new(sync.RWMutex)}
}

type clientReadWritAt struct {
	f *sftp.File
	m *sync.RWMutex
}

func (c *clientReadWritAt) WriteAt(p []byte, off int64) (n int, err error) {
	c.m.Lock()
	defer c.m.Unlock()
	_, _ = c.f.Seek(off, 0)
	return c.f.Write(p)
}

func CreateUsrftp(h string) *uConn {
	Uc := uConn{
		Addr:     h,
		Dirs:     map[string]os.FileInfo{},
		modeTime: time.Now().UTC(),
		fc:       map[string]*fCon{},
	}
	return &Uc
}

type sftpHandler struct {
	*uConn
	Hosts []config.Host
}

func (s *sftpHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		if r.Filepath == "/" {
			Files := make(listerat, 0, len(s.Hosts))
			// for i := 0; i < len(s.conf.Hosts); i++ {
			// 	Files = append(Files, &loFileInfo{dirName: s.conf.Hosts[i].Name})
			// }
			for _, v := range s.Hosts {
				Files = append(Files, &loFileInfo{dirName: v.Name})
			}
			return Files, nil
		} else {
			res, err := ReadDir(*s, r.Filepath)
			Files := make(listerat, 0, len(res))
			for i := 0; i < len(res); i++ {
				Files = append(Files, &reFileInfo{f: res[i]})
			}
			return Files, err
		}
	case "Stat":
		f, err := Stat(*s, r.Filepath)
		return listerat([]os.FileInfo{f}), err
	case "Readlink":
		logger.Debug("Readlink method", r.Filepath)
		r, err := ReadLink(*s, r.Filepath)
		return listerat([]os.FileInfo{&loFileInfo{dirName: r, linkName: "link"}}), err
	}
	return nil, sftp.ErrSshFxOpUnsupported
}

func (s *sftpHandler) Filecmd(r *sftp.Request) (err error) {
	logger.Debug("File cmd: ", r.Filepath)
	if r.Filepath == "/" {
		return sftp.ErrSshFxPermissionDenied
	}
	switch r.Method {
	case "Rename":
		return Rename(*s, r.Filepath, r.Target)
	case "Rmdir":
		return Rmdir(*s, r.Filepath)
	case "Remove":
		return Remove(*s, r.Filepath)
	case "Mkdir":
		return Mkdir(*s, r.Filepath)
	case "Symlink":
		return Symlink(*s, r.Filepath, r.Target)
	default:
		return
	}
}

func (s *sftpHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	logger.Debug("File write: ", r.Filepath)
	if r.Filepath == "/" {
		return nil, sftp.ErrSshFxPermissionDenied
	}
	f, err := Create(*s, r.Filepath)
	if err != nil {
		return nil, err
	}
	go func() {
		<-r.Context().Done()
		if err := f.Close(); err != nil {
			logger.Errorf("Remote sftp file %s close err: %s", r.Filepath, err)
		}
		logger.Infof("Sftp file write %s done", r.Filepath)
	}()
	return f, err
}

func (s *sftpHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	logger.Debug("File read: ", r.Filepath)
	if r.Filepath == "/" {
		return nil, sftp.ErrSshFxPermissionDenied
	}
	f, err := Open(*s, r.Filepath)
	if err != nil {
		return nil, err
	}
	go func() {
		<-r.Context().Done()
		if err := f.Close(); err != nil {
			logger.Errorf("Remote sftp file %s close err: %s", r.Filepath, err)
		}
		logger.Infof("Sftp file write %s done", r.Filepath)
	}()
	return f, err
}
func (s *sftpHandler) Close() {
	for c := range s.uConn.fc {
		s.uConn.fc[c].Close()
	}
}

type loFileInfo struct {
	dirName  string
	linkName string
}

func (f *loFileInfo) Name() string {
	return f.dirName
}
func (f *loFileInfo) Size() int64 { return 0 }
func (f *loFileInfo) Mode() os.FileMode {
	r := os.FileMode(0755) | os.ModeDir
	if f.linkName != "" {
		r = os.FileMode(0777) | os.ModeSymlink
	}
	return r
}
func (f *loFileInfo) ModTime() time.Time { return time.Now().UTC() }
func (f *loFileInfo) IsDir() bool        { return true }

func (*loFileInfo) Sys() interface{} {
	return nil
}

type reFileInfo struct {
	f os.FileInfo
}

func (r *reFileInfo) Name() string {
	return r.f.Name()
}
func (r *reFileInfo) Size() int64 { return r.f.Size() }
func (r *reFileInfo) Mode() os.FileMode {
	return r.f.Mode()
}
func (r *reFileInfo) ModTime() time.Time { return r.f.ModTime() }
func (r *reFileInfo) IsDir() bool        { return r.f.IsDir() }
func (r *reFileInfo) Sys() interface{} {
	return r.f.Sys()
}

type fCon struct {
	c *sftp.Client
	h string
}

func (s *fCon) Close() {
	if s.c == nil {
		return
	}
	_ = s.c.Close()
}

func RedirAll(c *sftp.Client, repath string) error {
	var err error
	var files []os.FileInfo
	files, err = c.ReadDir(repath)
	if err != nil {
		return err
	}
	for _, item := range files {
		jPath := filepath.Join(repath, item.Name())

		if item.IsDir() {
			err = RedirAll(c, jPath)
			if err != nil {
				return err
			}
			continue
		}
		err = c.Remove(jPath)
		if err != nil {
			return err
		}
	}
	return c.RemoveDirectory(repath)
}

func Mkdir(s sftpHandler, fpath string) error {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return err
	}
	return fconn.c.MkdirAll(repath)
}

func Remove(s sftpHandler, fpath string) error {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return err
	}
	return fconn.c.Remove(repath)
}
func Rmdir(s sftpHandler, fpath string) error {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return err
	}
	return RedirAll(fconn.c, repath)
}

func Symlink(s sftpHandler, lod string, new string) error {
	h, err := getRemotHost(s, lod)
	if err != nil {
		return err
	}
	fconn1, lodp, err := GetSftpCon(s, h, lod)
	if err != nil {
		return err
	}
	fconn2, newp, err := GetSftpCon(s, h, new)
	if err != nil {
		return err
	}
	if fconn1 != fconn2 {
		return sftp.ErrSshFxOpUnsupported
	}
	return fconn1.c.Symlink(lodp, newp)
}

func Rename(s sftpHandler, lod string, new string) error {
	h, err := getRemotHost(s, lod)
	if err != nil {
		return err
	}
	fconn1, lodp, err := GetSftpCon(s, h, lod)
	if err != nil {
		return err
	}
	fconn2, newp, err := GetSftpCon(s, h, new)
	if err != nil {
		return err
	}
	if fconn1 != fconn2 {
		return sftp.ErrSshFxOpUnsupported
	}
	return fconn1.c.Rename(lodp, newp)
}
func Create(s sftpHandler, fpath string) (f *sftp.File, err error) {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return f, err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return f, err
	}
	return fconn.c.Create(repath)
}
func Open(s sftpHandler, fpath string) (f *sftp.File, err error) {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return f, err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return f, err
	}
	return fconn.c.Open(repath)
}

func Stat(s sftpHandler, fpath string) (dirs os.FileInfo, err error) {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return dirs, err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return dirs, err
	}
	return fconn.c.Stat(repath)
}

func ReadDir(s sftpHandler, fpath string) (dirs []os.FileInfo, err error) {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return dirs, err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return dirs, err
	}
	return fconn.c.ReadDir(repath)
}

func ReadLink(s sftpHandler, fpath string) (r string, err error) {
	h, err := getRemotHost(s, fpath)
	if err != nil {
		return r, err
	}
	fconn, repath, err := GetSftpCon(s, h, fpath)
	if err != nil {
		return r, err
	}
	return fconn.c.ReadLink(repath)
}

func GetSftpCon(s sftpHandler, H config.Host, fpath string) (fconn *fCon, rpath string, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	var ok bool
	fpath = strings.TrimPrefix(strings.TrimPrefix(fpath, "/"), H.Name)
	fconn, ok = s.uConn.fc[H.Name]
	if !ok {
		fconn, err = CreateSftpConn(H)
		if err != nil {
			logger.Errorf("Create Sftp Client err: %s", err.Error())
			return nil, "", err
		}
		// logger.Debug(enddir, fconn, s.uConn.fc)
		s.uConn.fc[H.Name] = fconn
	}
	switch strings.ToLower(H.Ftpdir) {
	case "home", "~", "":
		rpath = filepath.Join(fconn.h, strings.TrimPrefix(fpath, "/"))
	default:
		if strings.Index(H.Ftpdir, "/") != 0 {
			H.Ftpdir = fmt.Sprintf("/%s", H.Ftpdir)
		}
		rpath = filepath.Join(H.Ftpdir, strings.TrimPrefix(fpath, "/"))
	}
	if runtime.GOOS == "windows" {
		rpath = strings.ReplaceAll(rpath, "\\", "/")
	}
	return
}

func CreateSftpConn(h config.Host) (*fCon, error) {
	Sc, err := proxy.CreateSSHClient(h)
	if err != nil {
		logger.Debug(err)
		logger.Errorf("Create SSH client err: %s", err)
		return nil, err
	}
	sess, err := Sc.AcquireSession()
	if err != nil {
		logger.Errorf("SSH client(%s) start sftp client session err %s", Sc, err)
		_ = Sc.Close()
		return nil, err
	}
	sftpClient, err := GetSftpClient(sess)
	if err != nil {
		logger.Errorf("SSH client(%s) start sftp conn err %s", Sc, err)
		_ = sess.Close()
		Sc.ReleaseSession(sess)
		return nil, err
	}
	go func() {
		_ = sftpClient.Wait()
		Sc.ReleaseSession(sess)
		logger.Infof("ssh client(%s) for SFTP release", Sc)
	}()
	HP, err := sftpClient.Getwd()
	if err != nil {
		logger.Errorf("SSH client sftp (%s) get home path err %s", Sc, err)
		_ = sftpClient.Close()
		return nil, err
	}
	logger.Infof("start sftp client session success")
	return &fCon{c: sftpClient, h: HP}, err
}

func GetSftpClient(sess *gossh.Session) (*sftp.Client, error) {
	if err := sess.RequestSubsystem("sftp"); err != nil {
		return nil, err
	}
	pw, err := sess.StdinPipe()
	if err != nil {
		return nil, err
	}
	pr, err := sess.StdoutPipe()
	if err != nil {
		return nil, err
	}
	client, err := sftp.NewClientPipe(pr, pw)
	if err != nil {
		return nil, err
	}
	return client, err
}

func getRemotHost(s sftpHandler, fpath string) (h config.Host, err error) {
	fp := strings.Split(strings.TrimPrefix(fpath, "/"), "/")
	enddir := fp[0]
	for _, v := range s.Hosts {
		if v.Name == enddir {
			h = v
			break
		}
	}
	if (h == config.Host{}) {
		logger.Debug("获取到hosts出错.", h)
		return h, filepath.ErrBadPattern
	}
	return
}
