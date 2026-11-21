// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Korean (`ko`).
class AppLocalizationsKo extends AppLocalizations {
  AppLocalizationsKo([String locale = 'ko']) : super(locale);

  @override
  String get appTitle => 'AVIBE';

  @override
  String get tabHome => '홈';

  @override
  String get tabSearch => '검색';

  @override
  String get tabPlaylists => '플리';

  @override
  String get tabPlayer => '재생';

  @override
  String get tabStore => '매장';

  @override
  String get sessionExpired => '세션이 만료되었습니다. 다시 로그인하세요.';

  @override
  String get loginTitle => 'AVIBE 로그인';

  @override
  String get loginEmail => '이메일';

  @override
  String get loginPassword => '비밀번호 (8자 이상)';

  @override
  String get loginTotp => '2단계 인증 코드';

  @override
  String get loginTotpHint => '인증 앱 6자리 또는 복구 코드';

  @override
  String get loginSubmit => '로그인';

  @override
  String get loginSubmitTotp => '인증하고 로그인';

  @override
  String get loginCreateAccount => '계정 만들기';

  @override
  String get loginHaveAccount => '이미 계정이 있어요';

  @override
  String get loginNeedAgeConfirm => '만 14세 이상 확인이 필요합니다.';

  @override
  String get loginTotpPrompt => '인증 앱의 6자리 코드를 입력하세요.';

  @override
  String get loginFailed => '인증에 실패했습니다.';

  @override
  String get loginOrSocial => '또는 소셜 계정으로 계속하기';

  @override
  String loginSocialFailed(String provider) {
    return '$provider 로그인에 실패했습니다.';
  }

  @override
  String get loginSocialCancelled => '소셜 로그인이 취소되었습니다.';

  @override
  String get loginSocialReturnFailed => '앱 복귀 처리에 실패했습니다. 다시 시도하세요.';

  @override
  String get providerGoogle => 'Google';

  @override
  String get providerKakao => '카카오';

  @override
  String get providerApple => 'Apple';

  @override
  String get providerNaver => '네이버';

  @override
  String get providerLine => 'LINE';

  @override
  String get signupTitle => 'AVIBE 회원가입';

  @override
  String get signupDisplayName => '표시 이름';

  @override
  String get signupBirthYear => '출생연도 (선택)';

  @override
  String get signupBirthYearHint => '예: 1995';

  @override
  String get signupInviteCode => '초대 코드 (필요 시)';

  @override
  String get signupAgeConfirm => '만 14세 이상입니다. (필수, 만 14세 미만은 가입 불가)';

  @override
  String get signupSubmit => '회원가입';

  @override
  String get signupBackToLogin => '로그인으로 돌아가기';

  @override
  String get ageGateTitle => '만 14세 이상 확인';

  @override
  String get ageGateDesc => '소셜 계정으로 처음 로그인했습니다. 스트리밍·결제를 이용하려면 연령 확인이 필요합니다.';

  @override
  String get ageGateConfirm => '만 14세 이상입니다. (필수)';

  @override
  String get ageGateBirthYear => '출생연도 (선택)';

  @override
  String get ageGateBirthYearHint => '예: 1995';

  @override
  String get ageGateFailed => '연령 확인에 실패했습니다.';

  @override
  String get ageGateLogout => '로그아웃';

  @override
  String get ageGateStart => '확인하고 시작하기';

  @override
  String get settingsTitle => '설정';

  @override
  String get settingsLanguage => '언어';

  @override
  String get settingsLanguageSaved => '언어가 저장되었습니다.';

  @override
  String get settingsLanguageFailed => '언어 저장에 실패했습니다.';

  @override
  String get settingsServerNote => '로그인하면 서버 언어 설정과 동기화됩니다.';

  @override
  String get settingsLogout => '로그아웃';

  @override
  String get settingsLegal => '약관 및 안내';

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
  String get homeEmpty => '아직 공개된 곡이 없습니다.';

  @override
  String get homeFailed => '피드를 불러오지 못했습니다.';

  @override
  String get searchTitle => '검색';

  @override
  String get searchHint => '곡 제목 또는 AI 제공자로 검색';

  @override
  String get searchEmpty => '검색 결과가 없습니다.';

  @override
  String get searchFailed => '검색에 실패했습니다.';

  @override
  String get playlistTitle => '플레이리스트';

  @override
  String get playlistEmpty => '플레이리스트가 없습니다. + 버튼으로 만드세요.';

  @override
  String get playlistPublic => '공개';

  @override
  String get playlistPrivate => '비공개';

  @override
  String get playlistRename => '이름 변경';

  @override
  String get playlistDelete => '삭제';

  @override
  String get playlistListFailed => '목록을 불러오지 못했습니다.';

  @override
  String get playlistNew => '새 플레이리스트';

  @override
  String get playlistEdit => '플레이리스트 수정';

  @override
  String get playlistNameLabel => '제목';

  @override
  String get playlistCancel => '취소';

  @override
  String get playlistSave => '저장';

  @override
  String get playlistSaveFailed => '저장에 실패했습니다.';

  @override
  String get playlistDeleteFailed => '삭제에 실패했습니다.';

  @override
  String get playlistSaveOrder => '순서 저장';

  @override
  String get playlistAddTrack => '곡 추가';

  @override
  String playlistDetailEmpty(String title) {
    return '\'$title\'에 곡이 없습니다.\n+ 버튼으로 추가하세요.';
  }

  @override
  String get playlistPlay => '재생';

  @override
  String get playlistDetailFailed => '상세를 불러오지 못했습니다.';

  @override
  String get playlistOrderFailed => '순서 저장에 실패했습니다.';

  @override
  String get playlistClose => '닫기';

  @override
  String get playlistAddFailed => '추가에 실패했습니다.';

  @override
  String get playerTitle => '재생 중';

  @override
  String get playerReportBlock => '신고 / 차단';

  @override
  String get playerTabWaveform => '파형';

  @override
  String get playerTabLyrics => '가사';

  @override
  String get playerTabAi => 'AI·라이선스';

  @override
  String get playerNoTrack => '재생 중인 곡이 없습니다.';

  @override
  String playerError(String detail) {
    return '플레이어 오류: $detail';
  }

  @override
  String get playerReportReasonTitle => '신고 사유 선택';

  @override
  String get playerBlockArtist => '이 아티스트(파트너) 차단 및 큐에서 제외';

  @override
  String get playerViewBlocks => '내 차단 내역 보기';

  @override
  String get playerBlockReasonDefault => '모바일 앱 사용자 직접 아티스트 차단';

  @override
  String get playerArtistBlocked => '아티스트가 차단되어 재생 목록에서 즉시 제외되었습니다.';

  @override
  String get playerBlockFailed => '아티스트 차단에 실패했습니다.';

  @override
  String get playerReportTitle => '신고하기';

  @override
  String get playerReportDetailLabel => '상세 설명 (선택)';

  @override
  String get playerReportDetailHint => '권리 침해 내용을 설명해주세요';

  @override
  String get playerCancel => '취소';

  @override
  String get playerReport => '신고';

  @override
  String playerReportReceived(String status) {
    return '신고가 접수되었습니다 ($status).';
  }

  @override
  String get playerReportFailed => '신고에 실패했습니다.';

  @override
  String get playerWaveformTitle => '오디오 파형 (Waveform)';

  @override
  String get playerInstrumental => '연주곡(Instrumental) 음원입니다.';

  @override
  String playerNoLyrics(String provider) {
    return '등록된 가사가 없습니다.\n이 곡은 $provider AI 모델로 생성된 음원입니다.';
  }

  @override
  String playerPromptLabel(String prompt) {
    return '생성 프롬프트: $prompt';
  }

  @override
  String get playerTrackFailed => '곡 정보를 불러오지 못했습니다.';

  @override
  String get playerAiLabel => 'AI 생성 여부';

  @override
  String get playerAiYes => 'AI 생성 음원';

  @override
  String get playerAiNo => '일반 음원';

  @override
  String get playerProviderLabel => 'AI 제공자';

  @override
  String get playerDisclosureLabel => '고지 문구';

  @override
  String get playerNoDisclosure => '별도 고지 없음';

  @override
  String get playerBitrateLabel => '비트레이트';

  @override
  String get playerKeyLabel => '키';

  @override
  String get playerPlayCountLabel => '재생 수';

  @override
  String get playerStatusLabel => '상태';

  @override
  String get playerAiNotice =>
      '이 음원은 AI 생성물 표시 의무에 따라 제공자와 생성 여부가 공개됩니다. 상업적 이용 범위는 제공자 라이선스 정책과 플랫폼 이용약관(TOS)을 따릅니다.';

  @override
  String get playerViewTerms => '이용약관 보기';

  @override
  String get legalTitle => '약관 및 안내';

  @override
  String get legalAiTitle => 'AI 생성물 안내';

  @override
  String get legalAiDesc => 'AI 음원 표시 및 라이선스 고지';

  @override
  String get legalBlocksTitle => '내 차단 내역';

  @override
  String get legalBlocksDesc => '신고 스트라이크에 따른 이용 제한';

  @override
  String get legalWithdrawalTitle => '회원 탈퇴 안내';

  @override
  String get legalTermsHeader => '이용약관';

  @override
  String get legalRequired => '필수';

  @override
  String legalVersion(String version) {
    return '버전 $version';
  }

  @override
  String get legalConsentRequired => ' · 필수 동의';

  @override
  String get legalConsent => '동의하기';

  @override
  String get legalConsentDone => '동의가 기록되었습니다.';

  @override
  String get legalConsentFailed => '동의 처리에 실패했습니다.';

  @override
  String get legalLoadFailed => '약관을 불러오지 못했습니다.';

  @override
  String get legalAiHeading => 'AI 음원 표시 안내';

  @override
  String get legalAiP1 =>
      '이 플랫폼의 모든 음원은 AI 생성 여부가 표시됩니다. 재생 화면의 \'AI·라이선스\' 탭에서 제공자(SUNO, UDIO, AIVA 등), 고지 문구, 비트레이트를 확인할 수 있습니다.';

  @override
  String get legalAiP2 =>
      'AI 음원의 상업적 이용 범위는 각 제공자의 라이선스 정책과 플랫폼 이용약관을 따릅니다. 무단 보이스 클론, 무단 샘플 사용이 의심되는 곡은 재생 화면의 신고 메뉴로 제보할 수 있습니다.';

  @override
  String get legalViewTos => '이용약관(TOS) 보기';

  @override
  String get legalNoBlocks => '차단 내역이 없습니다.';

  @override
  String legalStrikeItem(String number, String reason) {
    return '스트라이크 $number · $reason';
  }

  @override
  String get legalNoExpiry => '만료일 없음';

  @override
  String legalExpires(String date) {
    return '만료: $date';
  }

  @override
  String get legalBlocksFailed => '차단 내역을 불러오지 못했습니다.';

  @override
  String get legalWithdrawalHeading => '탈퇴 전 확인하세요';

  @override
  String get legalWithdrawalBody =>
      '· 플레이리스트와 재생 기록이 즉시 비활성화 및 삭제 처리됩니다.\n· 진행 중인 신고·분쟁이 있으면 관련 법령에 따라 보존 후 파기됩니다.\n· 아래 [즉시 회원 탈퇴] 버튼을 누르면 별도 고객센터 문의 없이 앱 내에서 즉시 탈퇴됩니다.';

  @override
  String get legalLogout => '로그아웃';

  @override
  String get legalWithdrawNow => '즉시 회원 탈퇴';

  @override
  String get storeTitle => '매장 모드';

  @override
  String get storeRefreshQueue => '큐 새로고침';

  @override
  String get storeUnpair => '페어링 해제';

  @override
  String get storeUnpairTitle => '페어링 해제';

  @override
  String get storeUnpairBody => '이 태블릿의 매장 페어링을 해제하고 재생을 중지합니다.';

  @override
  String get storeCancel => '취소';

  @override
  String get storeUnpairConfirm => '해제';

  @override
  String get storePairGuide => '파트너 콘솔의 매장 관리에서 발급된 8자리 페어링 코드를 입력하세요.';

  @override
  String get storePairCodeLabel => '페어링 코드';

  @override
  String get storePairCodeHint => '예: A1B2C3D4';

  @override
  String get storePair => '페어링';

  @override
  String get storePairFailed => '페어링에 실패했습니다.';

  @override
  String storeOpenNow(String business) {
    return '$business · 영업 중';
  }

  @override
  String storeOpenNowPlaylist(String business, String playlist) {
    return '$business · 영업 중 · $playlist';
  }

  @override
  String storeClosed(String business, String reason) {
    return '$business · $reason';
  }

  @override
  String get storeQueueEmpty => '재생할 곡이 없습니다.';

  @override
  String get storeNoSelection => '선택된 곡 없음';

  @override
  String get storePrev => '이전 곡';

  @override
  String get storePause => '일시정지';

  @override
  String get storePlay => '재생';

  @override
  String get storeNext => '다음 곡';

  @override
  String get errNotStreamable => '재생할 수 없는 곡입니다 (차단 또는 심사 중). 다음 곡으로 이동합니다.';

  @override
  String get errNotFound => '요청한 항목을 찾을 수 없습니다.';

  @override
  String get errAuth => '로그인이 필요하거나 인증이 만료되었습니다.';

  @override
  String get errInactive => '비활성화된 계정입니다. 고객센터에 문의하세요.';

  @override
  String get errForbidden => '권한이 없습니다.';

  @override
  String get errRateLimited => '요청이 너무 많습니다. 잠시 후 다시 시도하세요.';

  @override
  String get errEmailTaken => '이미 등록된 이메일입니다.';

  @override
  String get errTotpRequired => '2단계 인증 코드가 필요합니다.';

  @override
  String get errAgeConfirm => '만 14세 이상 확인이 필요합니다. 프로필에서 연령을 확인하세요.';

  @override
  String get errConsent => '필수 약관 동의가 필요합니다. 약관 화면에서 동의하세요.';

  @override
  String get errTotpInvalid => '인증 코드가 올바르지 않습니다. 다시 입력하세요.';

  @override
  String get errLoginLocked => '로그인 시도가 너무 많습니다. 15분 후 다시 시도하세요.';

  @override
  String get errProviderNotAllowed => '허용 목록에 없는 제공자입니다. 허용 제휴 제공자를 선택하세요.';

  @override
  String errInvalidInput(String detail) {
    return '입력값이 올바르지 않습니다: $detail';
  }

  @override
  String get errGeneric => '일시적인 오류가 발생했습니다.';

  @override
  String get errNetwork => '네트워크 연결에 실패했습니다.';

  @override
  String get errRequest => '요청에 실패했습니다.';

  @override
  String get reasonAiPolicy => 'AI 정책 위반';

  @override
  String get reasonVoiceClone => '무단 보이스 클론';

  @override
  String get reasonSample => '무단 샘플 사용';

  @override
  String get reasonOther => '기타';

  @override
  String get certTitle => '인증서 확인';

  @override
  String get certNumber => '인증서 번호';

  @override
  String get certIssuedAt => '발급일';

  @override
  String get certPlanTier => '요금제';

  @override
  String get certGates => '검증 게이트';

  @override
  String get certGatesPassed => '통과';

  @override
  String get certGatesFailed => '미통과';

  @override
  String get certNotFound => '인증서를 찾을 수 없습니다.';

  @override
  String get certLoadFailed => '인증서를 불러오지 못했습니다.';

  @override
  String get playerCertificateLabel => '인증서';

  @override
  String get completeEmailTitle => '이메일 입력';

  @override
  String get completeEmailSubtitle => '소셜 계정에서 이메일을 제공하지 않아 추가 입력이 필요합니다.';

  @override
  String get completeEmailHint => 'you@example.com';

  @override
  String get completeEmailSubmit => '계속하기';

  @override
  String get completeEmailBackToLogin => '로그인으로 돌아가기';

  @override
  String get completeEmailInvalid => '올바른 이메일 주소를 입력하세요.';

  @override
  String get completeEmailFailed => '제출에 실패했습니다.';

  @override
  String get completeEmailPendingTitle => '인증 메일을 보냈습니다';

  @override
  String completeEmailPendingBody(String email) {
    return '$email 주소로 인증 메일을 보냈습니다. 메일의 링크를 누르면 가입이 완료됩니다.';
  }
}
