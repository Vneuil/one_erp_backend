package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
)

var monthNames = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
var dayNames = [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
var romans = [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// idDate formats YYYY-MM-DD as "10 Maret 2026"; anything else is returned as is.
func idDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%d %s %d", t.Day(), monthNames[t.Month()-1], t.Year())
}

// idDayDate formats YYYY-MM-DD as "Selasa, 10 Maret 2026".
func idDayDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return dayNames[t.Weekday()] + ", " + idDate(s)
}

// rupiah formats an amount as "Rp 5.000.000".
func rupiah(v float64) string {
	n := int64(v + 0.5)
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, s[i])
	}
	return "Rp " + string(out)
}

type kind struct {
	code    string
	label   string
	subject string
}

var kinds = map[string]kind{
	domain.TypeContract:      {"SKK", "Surat Kontrak Kerja", "Perjanjian Kerja"},
	domain.TypeSummons:       {"SPG", "Surat Panggilan", "Surat Panggilan"},
	domain.TypeReprimand:     {"ST", "Surat Teguran", "Surat Teguran"},
	domain.TypeWarning:       {"SP", "Surat Peringatan", "Surat Peringatan"},
	domain.TypeTermination:   {"PHK", "Surat PHK", "Pemutusan Hubungan Kerja"},
	domain.TypeMemo:          {"MEMO", "Memo Internal", "Memo Internal"},
	domain.TypeOvertimeOrder: {"SPL", "Surat Perintah Lembur", "Surat Perintah Lembur"},
	domain.TypeMutation:      {"MUT", "Mutasi Karyawan", "Surat Mutasi Karyawan"},
}

var warningRoman = map[int]string{1: "I", 2: "II", 3: "III"}

// LetterNumber is "001/HRD-SP/III/2026": sequence within the type and year, type code, month in Roman numerals, year.
func LetterNumber(letterType string, seq int64, date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		t = time.Now()
	}
	return fmt.Sprintf("%03d/HRD-%s/%s/%d", seq, kinds[letterType].code, romans[t.Month()-1], t.Year())
}

func defaultSubject(l *domain.Letter) string {
	switch l.Type {
	case domain.TypeWarning:
		return "Surat Peringatan " + warningRoman[l.Level]
	case domain.TypeContract:
		switch l.Data.ContractType {
		case "PKWT":
			return "Perjanjian Kerja Waktu Tertentu (PKWT)"
		case "PKWTT":
			return "Perjanjian Kerja Waktu Tidak Tertentu (PKWTT)"
		case "Magang":
			return "Perjanjian Magang"
		}
	}
	return kinds[l.Type].subject
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func employeeLine(l *domain.Letter) string {
	return fmt.Sprintf("%s (NIP %s), %s pada %s", l.EmployeeName, l.NIP, orDash(l.Role), orDash(l.Department))
}

// RenderBody writes the standard Indonesian text of a letter from its data.
// The result is only a starting point: a draft's body can be edited before it is issued.
func RenderBody(l *domain.Letter) string {
	d := l.Data
	var p []string
	add := func(format string, a ...any) { p = append(p, fmt.Sprintf(format, a...)) }
	switch l.Type {
	case domain.TypeContract:
		add("Yang bertanda tangan di bawah ini, %s selaku %s %s (selanjutnya disebut \"Perusahaan\"), dan %s (selanjutnya disebut \"Karyawan\"), sepakat mengadakan perjanjian kerja dengan ketentuan sebagai berikut:", orDash(l.SignerName), orDash(l.SignerTitle), l.CompanyName, employeeLine(l))
		add("Pasal 1 - Jabatan dan Tempat Kerja\nKaryawan dipekerjakan sebagai %s pada %s, dengan tempat kerja di %s.", orDash(d.Position), orDash(l.Department), orDash(d.Workplace))
		switch d.ContractType {
		case "PKWT":
			add("Pasal 2 - Jangka Waktu\nPerjanjian kerja ini berlaku untuk waktu tertentu, sejak %s sampai dengan %s, dan berakhir demi hukum pada tanggal berakhirnya kecuali diperpanjang secara tertulis sesuai ketentuan perundang-undangan.", idDate(l.EffectiveDate), idDate(l.EndDate))
		case "Magang":
			add("Pasal 2 - Jangka Waktu\nProgram magang berlangsung sejak %s sampai dengan %s dan tidak menjamin pengangkatan sebagai karyawan.", idDate(l.EffectiveDate), idDate(l.EndDate))
		default:
			add("Pasal 2 - Jangka Waktu\nPerjanjian kerja ini berlaku untuk waktu tidak tertentu, terhitung sejak %s.", idDate(l.EffectiveDate))
		}
		if d.ProbationMonth > 0 {
			p[len(p)-1] += fmt.Sprintf(" Karyawan menjalani masa percobaan selama %d (%d) bulan.", d.ProbationMonth, d.ProbationMonth)
		}
		add("Pasal 3 - Upah\nPerusahaan membayar upah pokok sebesar %s per bulan, dibayarkan setiap bulan, setelah dikurangi kewajiban pajak penghasilan dan iuran yang diwajibkan peraturan perundang-undangan.", rupiah(d.Salary))
		add("Pasal 4 - Hak dan Kewajiban\nKaryawan wajib melaksanakan tugas dengan penuh tanggung jawab, menaati peraturan perusahaan dan perintah kerja yang sah, serta menjaga kerahasiaan informasi Perusahaan selama dan sesudah hubungan kerja. Perusahaan wajib membayar upah tepat waktu, menyediakan kondisi kerja yang layak, dan memenuhi hak Karyawan sesuai peraturan perundang-undangan.")
		add("Pasal 5 - Berakhirnya Hubungan Kerja\nHubungan kerja berakhir sesuai ketentuan perjanjian ini dan peraturan perundang-undangan ketenagakerjaan yang berlaku.")
		add("Pasal 6 - Penyelesaian Perselisihan\nPerselisihan yang timbul diselesaikan secara musyawarah untuk mufakat; bila tidak tercapai, diselesaikan melalui mekanisme penyelesaian perselisihan hubungan industrial sesuai peraturan perundang-undangan.")
		add("Demikian perjanjian kerja ini dibuat dan ditandatangani oleh kedua belah pihak di atas materai yang cukup, masing-masing mendapat satu lembar yang mempunyai kekuatan hukum yang sama.")
	case domain.TypeSummons:
		add("Dengan hormat,")
		add("Sehubungan dengan %s, kami meminta Saudara/i %s untuk hadir pada:", orDash(d.Reason), employeeLine(l))
		add("Hari/tanggal : %s\nPukul         : %s WIB\nTempat        : %s", idDayDate(d.MeetingDate), orDash(d.MeetingTime), orDash(d.Place))
		add("Mohon hadir tepat waktu. Apabila berhalangan, harap segera menghubungi bagian Human Resources sebelum waktu yang ditentukan.")
		add("Demikian surat panggilan ini disampaikan, atas perhatian dan kehadirannya kami ucapkan terima kasih.")
	case domain.TypeReprimand:
		add("Dengan hormat,")
		add("Berdasarkan pengamatan dan laporan yang kami terima, Saudara/i %s telah melakukan pelanggaran berikut:", employeeLine(l))
		add("%s", orDash(d.Violation))
		add("Perbuatan tersebut tidak sesuai dengan peraturan perusahaan. Melalui surat ini kami memberikan teguran agar Saudara/i memperbaiki sikap dan tidak mengulangi perbuatan tersebut.")
		if strings.TrimSpace(d.Consequence) != "" {
			add("%s", d.Consequence)
		} else {
			add("Apabila pelanggaran serupa terulang, Perusahaan akan menjatuhkan sanksi yang lebih berat berupa Surat Peringatan sesuai peraturan perusahaan.")
		}
		add("Demikian surat teguran ini disampaikan untuk diperhatikan.")
	case domain.TypeWarning:
		add("Dengan hormat,")
		add("Berdasarkan hasil pemeriksaan, Saudara/i %s telah melakukan pelanggaran berikut:", employeeLine(l))
		add("%s", orDash(d.Violation))
		add("Atas pelanggaran tersebut, Perusahaan memberikan Surat Peringatan %s. Surat peringatan ini berlaku selama %d bulan, sejak %s sampai dengan %s.", warningRoman[l.Level], d.ValidMonths, idDate(l.EffectiveDate), idDate(l.EndDate))
		switch {
		case strings.TrimSpace(d.Consequence) != "":
			add("%s", d.Consequence)
		case l.Level >= 3:
			add("Surat peringatan ini merupakan peringatan terakhir. Apabila Saudara/i kembali melakukan pelanggaran selama masa berlakunya, Perusahaan dapat melakukan pemutusan hubungan kerja sesuai peraturan perundang-undangan.")
		default:
			add("Apabila Saudara/i kembali melakukan pelanggaran selama masa berlaku surat peringatan ini, Perusahaan akan menjatuhkan sanksi yang lebih berat.")
		}
		add("Demikian surat peringatan ini disampaikan untuk diperhatikan dan dilaksanakan.")
	case domain.TypeTermination:
		add("Dengan hormat,")
		add("Sehubungan dengan %s, Perusahaan memberitahukan bahwa hubungan kerja dengan Saudara/i %s berakhir terhitung sejak hari terakhir bekerja pada tanggal %s.", orDash(d.TerminationReason), employeeLine(l), idDate(d.LastWorkDay))
		var rights []string
		if d.Severance > 0 {
			rights = append(rights, "uang pesangon "+rupiah(d.Severance))
		}
		if d.ServiceAward > 0 {
			rights = append(rights, "uang penghargaan masa kerja "+rupiah(d.ServiceAward))
		}
		if d.CompensationRights > 0 {
			rights = append(rights, "uang penggantian hak "+rupiah(d.CompensationRights))
		}
		if len(rights) > 0 {
			add("Perusahaan akan memenuhi hak Saudara/i berupa %s, sesuai peraturan perundang-undangan yang berlaku.", strings.Join(rights, ", "))
		} else {
			add("Hak-hak Saudara/i akan diselesaikan sesuai peraturan perundang-undangan yang berlaku.")
		}
		add("Sebelum hari terakhir bekerja, Saudara/i diminta menyelesaikan serah terima pekerjaan dan mengembalikan seluruh aset serta dokumen milik Perusahaan.")
		if strings.TrimSpace(d.Notes) != "" {
			add("%s", d.Notes)
		}
		add("Kami mengucapkan terima kasih atas kontribusi Saudara/i selama bekerja di Perusahaan.")
	case domain.TypeOvertimeOrder:
		add("Dengan ini Perusahaan memerintahkan karyawan yang tercantum di bawah untuk melaksanakan kerja lembur pada %s, pukul %s sampai dengan %s WIB.", idDayDate(d.WorkDate), orDash(d.StartTime), orDash(d.EndTime))
		add("Pekerjaan yang dilaksanakan:\n%s", orDash(d.Tasks))
		var names []string
		for i, r := range l.Recipients {
			names = append(names, fmt.Sprintf("%d. %s (NIP %s) - %s", i+1, r.Name, r.NIP, orDash(r.Department)))
		}
		add("Karyawan yang ditugaskan:\n%s", strings.Join(names, "\n"))
		add("Kerja lembur dilaksanakan atas perintah Perusahaan dan upah lembur dibayarkan sesuai ketentuan peraturan perundang-undangan yang berlaku.")
	case domain.TypeMutation:
		add("Dengan hormat,")
		add("Berdasarkan kebutuhan organisasi, Perusahaan memutuskan memutasikan Saudara/i %s (NIP %s) terhitung mulai tanggal %s, dengan rincian:", l.EmployeeName, l.NIP, idDate(l.EffectiveDate))
		add("Dari  : %s pada %s\nKe    : %s pada %s", orDash(d.FromRole), orDash(d.FromDepartment), orDash(firstNonEmpty(d.ToRole, d.FromRole)), orDash(firstNonEmpty(d.ToDepartment, d.FromDepartment)))
		add("Mutasi ini tidak mengurangi hak-hak Saudara/i yang telah diperoleh. Mohon melaksanakan serah terima pekerjaan dengan baik dan bekerja sesuai tugas pada penempatan yang baru.")
		add("Demikian surat mutasi ini disampaikan untuk dilaksanakan dengan penuh tanggung jawab.")
	}
	return strings.Join(p, "\n\n")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
