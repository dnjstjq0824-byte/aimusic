// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Thai (`th`).
class AppLocalizationsTh extends AppLocalizations {
  AppLocalizationsTh([String locale = 'th']) : super(locale);

  @override
  String get appTitle => 'AVIBE';

  @override
  String get tabHome => 'หน้าแรก';

  @override
  String get tabSearch => 'ค้นหา';

  @override
  String get tabPlaylists => 'เพลย์ลิสต์';

  @override
  String get tabPlayer => 'เล่น';

  @override
  String get tabStore => 'ร้านค้า';

  @override
  String get sessionExpired => 'เซสชันหมดอายุ กรุณาเข้าสู่ระบบอีกครั้ง';

  @override
  String get loginTitle => 'เข้าสู่ระบบ AVIBE';

  @override
  String get loginEmail => 'อีเมล';

  @override
  String get loginPassword => 'รหัสผ่าน (8 ตัวอักษรขึ้นไป)';

  @override
  String get loginTotp => 'รหัสยืนยันสองขั้นตอน';

  @override
  String get loginTotpHint => 'รหัส 6 หลักจากแอปหรือรหัสกู้คืน';

  @override
  String get loginSubmit => 'เข้าสู่ระบบ';

  @override
  String get loginSubmitTotp => 'ยืนยันและเข้าสู่ระบบ';

  @override
  String get loginCreateAccount => 'สร้างบัญชี';

  @override
  String get loginHaveAccount => 'ฉันมีบัญชีอยู่แล้ว';

  @override
  String get loginNeedAgeConfirm => 'ต้องยืนยันว่าคุณอายุ 14 ปีขึ้นไป';

  @override
  String get loginTotpPrompt => 'กรอกรหัส 6 หลักจากแอปยืนยันตัวตน';

  @override
  String get loginFailed => 'ยืนยันตัวตนล้มเหลว';

  @override
  String get loginOrSocial => 'หรือดำเนินการต่อด้วยบัญชีโซเชียล';

  @override
  String loginSocialFailed(String provider) {
    return 'เข้าสู่ระบบด้วย $provider ล้มเหลว';
  }

  @override
  String get loginSocialCancelled => 'ยกเลิกการเข้าสู่ระบบโซเชียลแล้ว';

  @override
  String get loginSocialReturnFailed => 'กลับเข้าแอปล้มเหลว กรุณาลองอีกครั้ง';

  @override
  String get providerGoogle => 'Google';

  @override
  String get providerKakao => 'Kakao';

  @override
  String get providerApple => 'Apple';

  @override
  String get providerNaver => 'Naver';

  @override
  String get providerLine => 'LINE';

  @override
  String get signupTitle => 'สร้างบัญชี AVIBE';

  @override
  String get signupDisplayName => 'ชื่อที่แสดง';

  @override
  String get signupBirthYear => 'ปีเกิด (ไม่บังคับ)';

  @override
  String get signupBirthYearHint => 'เช่น 1995';

  @override
  String get signupInviteCode => 'รหัสเชิญ (ถ้ามี)';

  @override
  String get signupAgeConfirm =>
      'ฉันอายุ 14 ปีขึ้นไป (บังคับ อายุต่ำกว่า 14 สมัครไม่ได้)';

  @override
  String get signupSubmit => 'สมัครสมาชิก';

  @override
  String get signupBackToLogin => 'กลับไปเข้าสู่ระบบ';

  @override
  String get ageGateTitle => 'ยืนยันอายุ 14 ปีขึ้นไป';

  @override
  String get ageGateDesc =>
      'คุณเข้าสู่ระบบด้วยบัญชีโซเชียลครั้งแรก ต้องยืนยันอายุก่อนใช้สตรีมมิงและการชำระเงิน';

  @override
  String get ageGateConfirm => 'ฉันอายุ 14 ปีขึ้นไป (บังคับ)';

  @override
  String get ageGateBirthYear => 'ปีเกิด (ไม่บังคับ)';

  @override
  String get ageGateBirthYearHint => 'เช่น 1995';

  @override
  String get ageGateFailed => 'ยืนยันอายุล้มเหลว';

  @override
  String get ageGateLogout => 'ออกจากระบบ';

  @override
  String get ageGateStart => 'ยืนยันและเริ่มต้น';

  @override
  String get settingsTitle => 'ตั้งค่า';

  @override
  String get settingsLanguage => 'ภาษา';

  @override
  String get settingsLanguageSaved => 'บันทึกภาษาแล้ว';

  @override
  String get settingsLanguageFailed => 'บันทึกภาษาล้มเหลว';

  @override
  String get settingsServerNote =>
      'เมื่อเข้าสู่ระบบ จะซิงก์กับการตั้งค่าภาษาบนเซิร์ฟเวอร์';

  @override
  String get settingsLogout => 'ออกจากระบบ';

  @override
  String get settingsLegal => 'ข้อกำหนดและประกาศ';

  @override
  String get langKo => '한국어';

  @override
  String get langEn => 'English';

  @override
  String get langJa => '日本語';

  @override
  String get langZh => '简体中文';

  @override
  String get langTh => 'ไทย';

  @override
  String get commonBeta => 'Beta';

  @override
  String get homeEmpty => 'ยังไม่มีเพลงที่เผยแพร่';

  @override
  String get homeFailed => 'โหลดฟีดไม่ได้';

  @override
  String get searchTitle => 'ค้นหา';

  @override
  String get searchHint => 'ค้นหาตามชื่อเพลงหรือผู้ให้บริการ AI';

  @override
  String get searchEmpty => 'ไม่พบผลการค้นหา';

  @override
  String get searchFailed => 'ค้นหาล้มเหลว';

  @override
  String get playlistTitle => 'เพลย์ลิสต์';

  @override
  String get playlistEmpty => 'ยังไม่มีเพลย์ลิสต์ แตะปุ่ม + เพื่อสร้าง';

  @override
  String get playlistPublic => 'สาธารณะ';

  @override
  String get playlistPrivate => 'ส่วนตัว';

  @override
  String get playlistRename => 'เปลี่ยนชื่อ';

  @override
  String get playlistDelete => 'ลบ';

  @override
  String get playlistListFailed => 'โหลดรายการไม่ได้';

  @override
  String get playlistNew => 'เพลย์ลิสต์ใหม่';

  @override
  String get playlistEdit => 'แก้ไขเพลย์ลิสต์';

  @override
  String get playlistNameLabel => 'ชื่อ';

  @override
  String get playlistCancel => 'ยกเลิก';

  @override
  String get playlistSave => 'บันทึก';

  @override
  String get playlistSaveFailed => 'บันทึกล้มเหลว';

  @override
  String get playlistDeleteFailed => 'ลบล้มเหลว';

  @override
  String get playlistSaveOrder => 'บันทึกลำดับ';

  @override
  String get playlistAddTrack => 'เพิ่มเพลง';

  @override
  String playlistDetailEmpty(String title) {
    return '\'$title\' ยังไม่มีเพลง\nแตะปุ่ม + เพื่อเพิ่ม';
  }

  @override
  String get playlistPlay => 'เล่น';

  @override
  String get playlistDetailFailed => 'โหลดรายละเอียดไม่ได้';

  @override
  String get playlistOrderFailed => 'บันทึกลำดับล้มเหลว';

  @override
  String get playlistClose => 'ปิด';

  @override
  String get playlistAddFailed => 'เพิ่มล้มเหลว';

  @override
  String get playerTitle => 'กำลังเล่น';

  @override
  String get playerReportBlock => 'รายงาน / บล็อก';

  @override
  String get playerTabWaveform => 'คลื่นเสียง';

  @override
  String get playerTabLyrics => 'เนื้อเพลง';

  @override
  String get playerTabAi => 'AI และสัญญาอนุญาต';

  @override
  String get playerNoTrack => 'ไม่มีเพลงที่กำลังเล่น';

  @override
  String playerError(String detail) {
    return 'ข้อผิดพลาดเครื่องเล่น: $detail';
  }

  @override
  String get playerReportReasonTitle => 'เลือกเหตุผลในการรายงาน';

  @override
  String get playerBlockArtist => 'บล็อกศิลปิน (พาร์ตเนอร์) นี้และนำออกจากคิว';

  @override
  String get playerViewBlocks => 'ดูรายการที่บล็อก';

  @override
  String get playerBlockReasonDefault => 'ผู้ใช้แอปมือถือบล็อกศิลปินโดยตรง';

  @override
  String get playerArtistBlocked => 'บล็อกศิลปินแล้วและนำออกจากรายการเล่นทันที';

  @override
  String get playerBlockFailed => 'บล็อกศิลปินล้มเหลว';

  @override
  String get playerReportTitle => 'รายงาน';

  @override
  String get playerReportDetailLabel => 'รายละเอียด (ไม่บังคับ)';

  @override
  String get playerReportDetailHint => 'อธิบายเนื้อหาที่ละเมิดสิทธิ์';

  @override
  String get playerCancel => 'ยกเลิก';

  @override
  String get playerReport => 'รายงาน';

  @override
  String playerReportReceived(String status) {
    return 'ได้รับรายงานแล้ว ($status)';
  }

  @override
  String get playerReportFailed => 'รายงานล้มเหลว';

  @override
  String get playerWaveformTitle => 'คลื่นเสียง (Waveform)';

  @override
  String get playerInstrumental => 'เพลงนี้เป็นเพลงบรรเลง (Instrumental)';

  @override
  String playerNoLyrics(String provider) {
    return 'ยังไม่มีเนื้อเพลง\nเพลงนี้สร้างด้วยโมเดล AI ของ $provider';
  }

  @override
  String playerPromptLabel(String prompt) {
    return 'พรอมต์การสร้าง: $prompt';
  }

  @override
  String get playerTrackFailed => 'โหลดข้อมูลเพลงไม่ได้';

  @override
  String get playerAiLabel => 'สร้างด้วย AI หรือไม่';

  @override
  String get playerAiYes => 'เพลงที่ AI สร้าง';

  @override
  String get playerAiNo => 'เพลงทั่วไป';

  @override
  String get playerProviderLabel => 'ผู้ให้บริการ AI';

  @override
  String get playerDisclosureLabel => 'ข้อความแจ้ง';

  @override
  String get playerNoDisclosure => 'ไม่มีข้อความแจ้งเพิ่มเติม';

  @override
  String get playerBitrateLabel => 'บิตเรต';

  @override
  String get playerKeyLabel => 'คีย์';

  @override
  String get playerPlayCountLabel => 'จำนวนครั้งที่เล่น';

  @override
  String get playerStatusLabel => 'สถานะ';

  @override
  String get playerAiNotice =>
      'เพลงนี้เปิดเผยผู้ให้บริการและสถานะ AI ตามหน้าที่แสดงเนื้อหาที่ AI สร้าง ขอบเขตการใช้เชิงพาณิชย์เป็นไปตามนโยบายสัญญาอนุญาตของผู้ให้บริการและข้อกำหนดการให้บริการ (TOS)';

  @override
  String get playerViewTerms => 'ดูข้อกำหนด';

  @override
  String get legalTitle => 'ข้อกำหนดและประกาศ';

  @override
  String get legalAiTitle => 'ประกาศเนื้อหา AI';

  @override
  String get legalAiDesc => 'การแสดงป้ายเพลง AI และการแจ้งสัญญาอนุญาต';

  @override
  String get legalBlocksTitle => 'รายการที่บล็อก';

  @override
  String get legalBlocksDesc => 'ข้อจำกัดการใช้งานจากการถูกรายงาน';

  @override
  String get legalWithdrawalTitle => 'คำแนะนำการถอนสมาชิก';

  @override
  String get legalTermsHeader => 'ข้อกำหนดการให้บริการ';

  @override
  String get legalRequired => 'บังคับ';

  @override
  String legalVersion(String version) {
    return 'เวอร์ชัน $version';
  }

  @override
  String get legalConsentRequired => ' · ต้องยินยอม';

  @override
  String get legalConsent => 'ยินยอม';

  @override
  String get legalConsentDone => 'บันทึกความยินยอมแล้ว';

  @override
  String get legalConsentFailed => 'บันทึกความยินยอมล้มเหลว';

  @override
  String get legalLoadFailed => 'โหลดข้อกำหนดไม่ได้';

  @override
  String get legalAiHeading => 'เกี่ยวกับการแสดงป้ายเพลง AI';

  @override
  String get legalAiP1 =>
      'เพลงทุกเพลงบนแพลตฟอร์มนี้แสดงสถานะการสร้างด้วย AI ตรวจสอบผู้ให้บริการ (SUNO, UDIO, AIVA ฯลฯ) ข้อความแจ้ง และบิตเรตได้ที่แท็บ \'AI และสัญญาอนุญาต\' ในหน้าจอเล่นเพลง';

  @override
  String get legalAiP2 =>
      'การใช้เพลง AI เชิงพาณิชย์เป็นไปตามนโยบายสัญญาอนุญาตของผู้ให้บริการและข้อกำหนดแพลตฟอร์ม หากสงสัยว่ามีการโคลนเสียงหรือใช้ตัวอย่างโดยไม่ได้รับอนุญาต ให้รายงานจากเมนูรายงานในหน้าจอเล่นเพลง';

  @override
  String get legalViewTos => 'ดูข้อกำหนดการให้บริการ (TOS)';

  @override
  String get legalNoBlocks => 'ไม่มีรายการที่บล็อก';

  @override
  String legalStrikeItem(String number, String reason) {
    return 'สไตรก์ $number · $reason';
  }

  @override
  String get legalNoExpiry => 'ไม่มีวันหมดอายุ';

  @override
  String legalExpires(String date) {
    return 'หมดอายุ: $date';
  }

  @override
  String get legalBlocksFailed => 'โหลดรายการที่บล็อกล้มเหลว';

  @override
  String get legalWithdrawalHeading => 'โปรดตรวจสอบก่อนถอนสมาชิก';

  @override
  String get legalWithdrawalBody =>
      '· เพลย์ลิสต์และประวัติการฟังจะถูกปิดใช้งานและลบออกทันที\n· หากมีรายงานหรือข้อพิพาทที่กำลังดำเนินการ จะเก็บรักษาตามกฎหมายที่เกี่ยวข้องแล้วทำลาย\n· แตะปุ่ม [ถอนสมาชิกทันที] ด้านล่างเพื่อถอนสมาชิกในแอปทันที โดยไม่ต้องติดต่อฝ่ายสนับสนุน';

  @override
  String get legalLogout => 'ออกจากระบบ';

  @override
  String get legalWithdrawNow => 'ถอนสมาชิกทันที';

  @override
  String get storeTitle => 'โหมดร้านค้า';

  @override
  String get storeRefreshQueue => 'รีเฟรชคิว';

  @override
  String get storeUnpair => 'ยกเลิกการจับคู่';

  @override
  String get storeUnpairTitle => 'ยกเลิกการจับคู่';

  @override
  String get storeUnpairBody =>
      'ยกเลิกการจับคู่แท็บเล็ตนี้กับร้านค้าและหยุดการเล่น';

  @override
  String get storeCancel => 'ยกเลิก';

  @override
  String get storeUnpairConfirm => 'ยกเลิก';

  @override
  String get storePairGuide =>
      'กรอกรหัสจับคู่ 8 หลักที่ออกในหน้าจัดการร้านค้าของคอนโซลพาร์ตเนอร์';

  @override
  String get storePairCodeLabel => 'รหัสจับคู่';

  @override
  String get storePairCodeHint => 'เช่น A1B2C3D4';

  @override
  String get storePair => 'จับคู่';

  @override
  String get storePairFailed => 'จับคู่ล้มเหลว';

  @override
  String storeOpenNow(String business) {
    return '$business · เปิดอยู่';
  }

  @override
  String storeOpenNowPlaylist(String business, String playlist) {
    return '$business · เปิดอยู่ · $playlist';
  }

  @override
  String storeClosed(String business, String reason) {
    return '$business · $reason';
  }

  @override
  String get storeQueueEmpty => 'ไม่มีเพลงให้เล่น';

  @override
  String get storeNoSelection => 'ไม่ได้เลือกเพลง';

  @override
  String get storePrev => 'เพลงก่อนหน้า';

  @override
  String get storePause => 'หยุดชั่วคราว';

  @override
  String get storePlay => 'เล่น';

  @override
  String get storeNext => 'เพลงถัดไป';

  @override
  String get errNotStreamable =>
      'เล่นเพลงนี้ไม่ได้ (ถูกบล็อกหรืออยู่ระหว่างตรวจสอบ) กำลังไปเพลงถัดไป';

  @override
  String get errNotFound => 'ไม่พบรายการที่ขอ';

  @override
  String get errAuth => 'ต้องเข้าสู่ระบบหรือการยืนยันหมดอายุ';

  @override
  String get errInactive => 'บัญชีนี้ถูกปิดใช้งาน ติดต่อฝ่ายสนับสนุน';

  @override
  String get errForbidden => 'ไม่มีสิทธิ์';

  @override
  String get errRateLimited => 'คำขอมากเกินไป กรุณาลองอีกครั้งในภายหลัง';

  @override
  String get errEmailTaken => 'อีเมลนี้ลงทะเบียนแล้ว';

  @override
  String get errTotpRequired => 'ต้องใช้รหัสยืนยันสองขั้นตอน';

  @override
  String get errAgeConfirm =>
      'ต้องยืนยันอายุ 14 ปีขึ้นไป กรุณายืนยันอายุในโปรไฟล์';

  @override
  String get errConsent =>
      'ต้องยินยอมข้อกำหนดบังคับ กรุณายินยอมที่หน้าข้อกำหนด';

  @override
  String get errTotpInvalid => 'รหัสไม่ถูกต้อง กรุณากรอกอีกครั้ง';

  @override
  String get errLoginLocked =>
      'พยายามเข้าสู่ระบบมากเกินไป ลองอีกครั้งใน 15 นาที';

  @override
  String get errProviderNotAllowed =>
      'ผู้ให้บริการนี้ไม่ได้อยู่ในรายการที่อนุญาต กรุณาเลือกผู้ให้บริการพาร์ตเนอร์ที่อนุญาต';

  @override
  String errInvalidInput(String detail) {
    return 'ข้อมูลไม่ถูกต้อง: $detail';
  }

  @override
  String get errGeneric => 'เกิดข้อผิดพลาดชั่วคราว';

  @override
  String get errNetwork => 'เชื่อมต่อเครือข่ายล้มเหลว';

  @override
  String get errRequest => 'คำขอล้มเหลว';

  @override
  String get reasonAiPolicy => 'ละเมิดนโยบาย AI';

  @override
  String get reasonVoiceClone => 'โคลนเสียงโดยไม่ได้รับอนุญาต';

  @override
  String get reasonSample => 'ใช้ตัวอย่างโดยไม่ได้รับอนุญาต';

  @override
  String get reasonOther => 'อื่น ๆ';

  @override
  String get certTitle => 'ตรวจสอบใบรับรอง';

  @override
  String get certNumber => 'หมายเลขใบรับรอง';

  @override
  String get certIssuedAt => 'วันที่ออก';

  @override
  String get certPlanTier => 'แผน';

  @override
  String get certGates => 'เกณฑ์ตรวจสอบ';

  @override
  String get certGatesPassed => 'ผ่าน';

  @override
  String get certGatesFailed => 'ไม่ผ่าน';

  @override
  String get certNotFound => 'ไม่พบใบรับรอง';

  @override
  String get certLoadFailed => 'โหลดใบรับรองไม่สำเร็จ';

  @override
  String get playerCertificateLabel => 'ใบรับรอง';

  @override
  String get completeEmailTitle => 'กรอกอีเมล';

  @override
  String get completeEmailSubtitle =>
      'บัญชีโซเชียลไม่ได้แชร์อีเมล จึงต้องขออีกขั้น';

  @override
  String get completeEmailHint => 'you@example.com';

  @override
  String get completeEmailSubmit => 'ดำเนินการต่อ';

  @override
  String get completeEmailBackToLogin => 'กลับไปล็อกอิน';

  @override
  String get completeEmailInvalid => 'กรุณากรอกอีเมลที่ถูกต้อง';

  @override
  String get completeEmailFailed => 'ส่งไม่สำเร็จ';

  @override
  String get completeEmailPendingTitle => 'ส่งจดหมายยืนยันแล้ว';

  @override
  String completeEmailPendingBody(String email) {
    return 'เราส่งจดหมายยืนยันไปที่ $email แล้ว เปิดลิงก์ในจดหมายเพื่อสมัครให้เสร็จสมบูรณ์';
  }
}
