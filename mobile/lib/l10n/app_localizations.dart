import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_ja.dart';
import 'app_localizations_ko.dart';
import 'app_localizations_th.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
      : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
    delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
  ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('ko'),
    Locale('en'),
    Locale('ja'),
    Locale('zh'),
    Locale('th')
  ];

  /// No description provided for @appTitle.
  ///
  /// In ko, this message translates to:
  /// **'AVIBE'**
  String get appTitle;

  /// No description provided for @tabHome.
  ///
  /// In ko, this message translates to:
  /// **'홈'**
  String get tabHome;

  /// No description provided for @tabSearch.
  ///
  /// In ko, this message translates to:
  /// **'검색'**
  String get tabSearch;

  /// No description provided for @tabPlaylists.
  ///
  /// In ko, this message translates to:
  /// **'플리'**
  String get tabPlaylists;

  /// No description provided for @tabPlayer.
  ///
  /// In ko, this message translates to:
  /// **'재생'**
  String get tabPlayer;

  /// No description provided for @tabStore.
  ///
  /// In ko, this message translates to:
  /// **'매장'**
  String get tabStore;

  /// No description provided for @sessionExpired.
  ///
  /// In ko, this message translates to:
  /// **'세션이 만료되었습니다. 다시 로그인하세요.'**
  String get sessionExpired;

  /// No description provided for @loginTitle.
  ///
  /// In ko, this message translates to:
  /// **'AVIBE 로그인'**
  String get loginTitle;

  /// No description provided for @loginEmail.
  ///
  /// In ko, this message translates to:
  /// **'이메일'**
  String get loginEmail;

  /// No description provided for @loginPassword.
  ///
  /// In ko, this message translates to:
  /// **'비밀번호 (8자 이상)'**
  String get loginPassword;

  /// No description provided for @loginTotp.
  ///
  /// In ko, this message translates to:
  /// **'2단계 인증 코드'**
  String get loginTotp;

  /// No description provided for @loginTotpHint.
  ///
  /// In ko, this message translates to:
  /// **'인증 앱 6자리 또는 복구 코드'**
  String get loginTotpHint;

  /// No description provided for @loginSubmit.
  ///
  /// In ko, this message translates to:
  /// **'로그인'**
  String get loginSubmit;

  /// No description provided for @loginSubmitTotp.
  ///
  /// In ko, this message translates to:
  /// **'인증하고 로그인'**
  String get loginSubmitTotp;

  /// No description provided for @loginCreateAccount.
  ///
  /// In ko, this message translates to:
  /// **'계정 만들기'**
  String get loginCreateAccount;

  /// No description provided for @loginHaveAccount.
  ///
  /// In ko, this message translates to:
  /// **'이미 계정이 있어요'**
  String get loginHaveAccount;

  /// No description provided for @loginNeedAgeConfirm.
  ///
  /// In ko, this message translates to:
  /// **'만 14세 이상 확인이 필요합니다.'**
  String get loginNeedAgeConfirm;

  /// No description provided for @loginTotpPrompt.
  ///
  /// In ko, this message translates to:
  /// **'인증 앱의 6자리 코드를 입력하세요.'**
  String get loginTotpPrompt;

  /// No description provided for @loginFailed.
  ///
  /// In ko, this message translates to:
  /// **'인증에 실패했습니다.'**
  String get loginFailed;

  /// No description provided for @loginOrSocial.
  ///
  /// In ko, this message translates to:
  /// **'또는 소셜 계정으로 계속하기'**
  String get loginOrSocial;

  /// No description provided for @loginSocialFailed.
  ///
  /// In ko, this message translates to:
  /// **'{provider} 로그인에 실패했습니다.'**
  String loginSocialFailed(String provider);

  /// No description provided for @loginSocialCancelled.
  ///
  /// In ko, this message translates to:
  /// **'소셜 로그인이 취소되었습니다.'**
  String get loginSocialCancelled;

  /// No description provided for @loginSocialReturnFailed.
  ///
  /// In ko, this message translates to:
  /// **'앱 복귀 처리에 실패했습니다. 다시 시도하세요.'**
  String get loginSocialReturnFailed;

  /// No description provided for @providerGoogle.
  ///
  /// In ko, this message translates to:
  /// **'Google'**
  String get providerGoogle;

  /// No description provided for @providerKakao.
  ///
  /// In ko, this message translates to:
  /// **'카카오'**
  String get providerKakao;

  /// No description provided for @providerApple.
  ///
  /// In ko, this message translates to:
  /// **'Apple'**
  String get providerApple;

  /// No description provided for @providerNaver.
  ///
  /// In ko, this message translates to:
  /// **'네이버'**
  String get providerNaver;

  /// No description provided for @providerLine.
  ///
  /// In ko, this message translates to:
  /// **'LINE'**
  String get providerLine;

  /// No description provided for @signupTitle.
  ///
  /// In ko, this message translates to:
  /// **'AVIBE 회원가입'**
  String get signupTitle;

  /// No description provided for @signupDisplayName.
  ///
  /// In ko, this message translates to:
  /// **'표시 이름'**
  String get signupDisplayName;

  /// No description provided for @signupBirthYear.
  ///
  /// In ko, this message translates to:
  /// **'출생연도 (선택)'**
  String get signupBirthYear;

  /// No description provided for @signupBirthYearHint.
  ///
  /// In ko, this message translates to:
  /// **'예: 1995'**
  String get signupBirthYearHint;

  /// No description provided for @signupInviteCode.
  ///
  /// In ko, this message translates to:
  /// **'초대 코드 (필요 시)'**
  String get signupInviteCode;

  /// No description provided for @signupAgeConfirm.
  ///
  /// In ko, this message translates to:
  /// **'만 14세 이상입니다. (필수, 만 14세 미만은 가입 불가)'**
  String get signupAgeConfirm;

  /// No description provided for @signupSubmit.
  ///
  /// In ko, this message translates to:
  /// **'회원가입'**
  String get signupSubmit;

  /// No description provided for @signupBackToLogin.
  ///
  /// In ko, this message translates to:
  /// **'로그인으로 돌아가기'**
  String get signupBackToLogin;

  /// No description provided for @ageGateTitle.
  ///
  /// In ko, this message translates to:
  /// **'만 14세 이상 확인'**
  String get ageGateTitle;

  /// No description provided for @ageGateDesc.
  ///
  /// In ko, this message translates to:
  /// **'소셜 계정으로 처음 로그인했습니다. 스트리밍·결제를 이용하려면 연령 확인이 필요합니다.'**
  String get ageGateDesc;

  /// No description provided for @ageGateConfirm.
  ///
  /// In ko, this message translates to:
  /// **'만 14세 이상입니다. (필수)'**
  String get ageGateConfirm;

  /// No description provided for @ageGateBirthYear.
  ///
  /// In ko, this message translates to:
  /// **'출생연도 (선택)'**
  String get ageGateBirthYear;

  /// No description provided for @ageGateBirthYearHint.
  ///
  /// In ko, this message translates to:
  /// **'예: 1995'**
  String get ageGateBirthYearHint;

  /// No description provided for @ageGateFailed.
  ///
  /// In ko, this message translates to:
  /// **'연령 확인에 실패했습니다.'**
  String get ageGateFailed;

  /// No description provided for @ageGateLogout.
  ///
  /// In ko, this message translates to:
  /// **'로그아웃'**
  String get ageGateLogout;

  /// No description provided for @ageGateStart.
  ///
  /// In ko, this message translates to:
  /// **'확인하고 시작하기'**
  String get ageGateStart;

  /// No description provided for @settingsTitle.
  ///
  /// In ko, this message translates to:
  /// **'설정'**
  String get settingsTitle;

  /// No description provided for @settingsLanguage.
  ///
  /// In ko, this message translates to:
  /// **'언어'**
  String get settingsLanguage;

  /// No description provided for @settingsLanguageSaved.
  ///
  /// In ko, this message translates to:
  /// **'언어가 저장되었습니다.'**
  String get settingsLanguageSaved;

  /// No description provided for @settingsLanguageFailed.
  ///
  /// In ko, this message translates to:
  /// **'언어 저장에 실패했습니다.'**
  String get settingsLanguageFailed;

  /// No description provided for @settingsServerNote.
  ///
  /// In ko, this message translates to:
  /// **'로그인하면 서버 언어 설정과 동기화됩니다.'**
  String get settingsServerNote;

  /// No description provided for @settingsLogout.
  ///
  /// In ko, this message translates to:
  /// **'로그아웃'**
  String get settingsLogout;

  /// No description provided for @settingsLegal.
  ///
  /// In ko, this message translates to:
  /// **'약관 및 안내'**
  String get settingsLegal;

  /// No description provided for @langKo.
  ///
  /// In ko, this message translates to:
  /// **'한국어'**
  String get langKo;

  /// No description provided for @langEn.
  ///
  /// In ko, this message translates to:
  /// **'English'**
  String get langEn;

  /// No description provided for @langJa.
  ///
  /// In ko, this message translates to:
  /// **'日本語'**
  String get langJa;

  /// No description provided for @langZh.
  ///
  /// In ko, this message translates to:
  /// **'简体中文'**
  String get langZh;

  /// No description provided for @langTh.
  ///
  /// In ko, this message translates to:
  /// **'ไทย'**
  String get langTh;

  /// No description provided for @commonBeta.
  ///
  /// In ko, this message translates to:
  /// **'Beta'**
  String get commonBeta;

  /// No description provided for @homeEmpty.
  ///
  /// In ko, this message translates to:
  /// **'아직 공개된 곡이 없습니다.'**
  String get homeEmpty;

  /// No description provided for @homeFailed.
  ///
  /// In ko, this message translates to:
  /// **'피드를 불러오지 못했습니다.'**
  String get homeFailed;

  /// No description provided for @searchTitle.
  ///
  /// In ko, this message translates to:
  /// **'검색'**
  String get searchTitle;

  /// No description provided for @searchHint.
  ///
  /// In ko, this message translates to:
  /// **'곡 제목 또는 AI 제공자로 검색'**
  String get searchHint;

  /// No description provided for @searchEmpty.
  ///
  /// In ko, this message translates to:
  /// **'검색 결과가 없습니다.'**
  String get searchEmpty;

  /// No description provided for @searchFailed.
  ///
  /// In ko, this message translates to:
  /// **'검색에 실패했습니다.'**
  String get searchFailed;

  /// No description provided for @playlistTitle.
  ///
  /// In ko, this message translates to:
  /// **'플레이리스트'**
  String get playlistTitle;

  /// No description provided for @playlistEmpty.
  ///
  /// In ko, this message translates to:
  /// **'플레이리스트가 없습니다. + 버튼으로 만드세요.'**
  String get playlistEmpty;

  /// No description provided for @playlistPublic.
  ///
  /// In ko, this message translates to:
  /// **'공개'**
  String get playlistPublic;

  /// No description provided for @playlistPrivate.
  ///
  /// In ko, this message translates to:
  /// **'비공개'**
  String get playlistPrivate;

  /// No description provided for @playlistRename.
  ///
  /// In ko, this message translates to:
  /// **'이름 변경'**
  String get playlistRename;

  /// No description provided for @playlistDelete.
  ///
  /// In ko, this message translates to:
  /// **'삭제'**
  String get playlistDelete;

  /// No description provided for @playlistListFailed.
  ///
  /// In ko, this message translates to:
  /// **'목록을 불러오지 못했습니다.'**
  String get playlistListFailed;

  /// No description provided for @playlistNew.
  ///
  /// In ko, this message translates to:
  /// **'새 플레이리스트'**
  String get playlistNew;

  /// No description provided for @playlistEdit.
  ///
  /// In ko, this message translates to:
  /// **'플레이리스트 수정'**
  String get playlistEdit;

  /// No description provided for @playlistNameLabel.
  ///
  /// In ko, this message translates to:
  /// **'제목'**
  String get playlistNameLabel;

  /// No description provided for @playlistCancel.
  ///
  /// In ko, this message translates to:
  /// **'취소'**
  String get playlistCancel;

  /// No description provided for @playlistSave.
  ///
  /// In ko, this message translates to:
  /// **'저장'**
  String get playlistSave;

  /// No description provided for @playlistSaveFailed.
  ///
  /// In ko, this message translates to:
  /// **'저장에 실패했습니다.'**
  String get playlistSaveFailed;

  /// No description provided for @playlistDeleteFailed.
  ///
  /// In ko, this message translates to:
  /// **'삭제에 실패했습니다.'**
  String get playlistDeleteFailed;

  /// No description provided for @playlistSaveOrder.
  ///
  /// In ko, this message translates to:
  /// **'순서 저장'**
  String get playlistSaveOrder;

  /// No description provided for @playlistAddTrack.
  ///
  /// In ko, this message translates to:
  /// **'곡 추가'**
  String get playlistAddTrack;

  /// No description provided for @playlistDetailEmpty.
  ///
  /// In ko, this message translates to:
  /// **'\'{title}\'에 곡이 없습니다.\n+ 버튼으로 추가하세요.'**
  String playlistDetailEmpty(String title);

  /// No description provided for @playlistPlay.
  ///
  /// In ko, this message translates to:
  /// **'재생'**
  String get playlistPlay;

  /// No description provided for @playlistDetailFailed.
  ///
  /// In ko, this message translates to:
  /// **'상세를 불러오지 못했습니다.'**
  String get playlistDetailFailed;

  /// No description provided for @playlistOrderFailed.
  ///
  /// In ko, this message translates to:
  /// **'순서 저장에 실패했습니다.'**
  String get playlistOrderFailed;

  /// No description provided for @playlistClose.
  ///
  /// In ko, this message translates to:
  /// **'닫기'**
  String get playlistClose;

  /// No description provided for @playlistAddFailed.
  ///
  /// In ko, this message translates to:
  /// **'추가에 실패했습니다.'**
  String get playlistAddFailed;

  /// No description provided for @playerTitle.
  ///
  /// In ko, this message translates to:
  /// **'재생 중'**
  String get playerTitle;

  /// No description provided for @playerReportBlock.
  ///
  /// In ko, this message translates to:
  /// **'신고 / 차단'**
  String get playerReportBlock;

  /// No description provided for @playerTabWaveform.
  ///
  /// In ko, this message translates to:
  /// **'파형'**
  String get playerTabWaveform;

  /// No description provided for @playerTabLyrics.
  ///
  /// In ko, this message translates to:
  /// **'가사'**
  String get playerTabLyrics;

  /// No description provided for @playerTabAi.
  ///
  /// In ko, this message translates to:
  /// **'AI·라이선스'**
  String get playerTabAi;

  /// No description provided for @playerNoTrack.
  ///
  /// In ko, this message translates to:
  /// **'재생 중인 곡이 없습니다.'**
  String get playerNoTrack;

  /// No description provided for @playerError.
  ///
  /// In ko, this message translates to:
  /// **'플레이어 오류: {detail}'**
  String playerError(String detail);

  /// No description provided for @playerReportReasonTitle.
  ///
  /// In ko, this message translates to:
  /// **'신고 사유 선택'**
  String get playerReportReasonTitle;

  /// No description provided for @playerBlockArtist.
  ///
  /// In ko, this message translates to:
  /// **'이 아티스트(파트너) 차단 및 큐에서 제외'**
  String get playerBlockArtist;

  /// No description provided for @playerViewBlocks.
  ///
  /// In ko, this message translates to:
  /// **'내 차단 내역 보기'**
  String get playerViewBlocks;

  /// No description provided for @playerBlockReasonDefault.
  ///
  /// In ko, this message translates to:
  /// **'모바일 앱 사용자 직접 아티스트 차단'**
  String get playerBlockReasonDefault;

  /// No description provided for @playerArtistBlocked.
  ///
  /// In ko, this message translates to:
  /// **'아티스트가 차단되어 재생 목록에서 즉시 제외되었습니다.'**
  String get playerArtistBlocked;

  /// No description provided for @playerBlockFailed.
  ///
  /// In ko, this message translates to:
  /// **'아티스트 차단에 실패했습니다.'**
  String get playerBlockFailed;

  /// No description provided for @playerReportTitle.
  ///
  /// In ko, this message translates to:
  /// **'신고하기'**
  String get playerReportTitle;

  /// No description provided for @playerReportDetailLabel.
  ///
  /// In ko, this message translates to:
  /// **'상세 설명 (선택)'**
  String get playerReportDetailLabel;

  /// No description provided for @playerReportDetailHint.
  ///
  /// In ko, this message translates to:
  /// **'권리 침해 내용을 설명해주세요'**
  String get playerReportDetailHint;

  /// No description provided for @playerCancel.
  ///
  /// In ko, this message translates to:
  /// **'취소'**
  String get playerCancel;

  /// No description provided for @playerReport.
  ///
  /// In ko, this message translates to:
  /// **'신고'**
  String get playerReport;

  /// No description provided for @playerReportReceived.
  ///
  /// In ko, this message translates to:
  /// **'신고가 접수되었습니다 ({status}).'**
  String playerReportReceived(String status);

  /// No description provided for @playerReportFailed.
  ///
  /// In ko, this message translates to:
  /// **'신고에 실패했습니다.'**
  String get playerReportFailed;

  /// No description provided for @playerWaveformTitle.
  ///
  /// In ko, this message translates to:
  /// **'오디오 파형 (Waveform)'**
  String get playerWaveformTitle;

  /// No description provided for @playerInstrumental.
  ///
  /// In ko, this message translates to:
  /// **'연주곡(Instrumental) 음원입니다.'**
  String get playerInstrumental;

  /// No description provided for @playerNoLyrics.
  ///
  /// In ko, this message translates to:
  /// **'등록된 가사가 없습니다.\n이 곡은 {provider} AI 모델로 생성된 음원입니다.'**
  String playerNoLyrics(String provider);

  /// No description provided for @playerPromptLabel.
  ///
  /// In ko, this message translates to:
  /// **'생성 프롬프트: {prompt}'**
  String playerPromptLabel(String prompt);

  /// No description provided for @playerTrackFailed.
  ///
  /// In ko, this message translates to:
  /// **'곡 정보를 불러오지 못했습니다.'**
  String get playerTrackFailed;

  /// No description provided for @playerAiLabel.
  ///
  /// In ko, this message translates to:
  /// **'AI 생성 여부'**
  String get playerAiLabel;

  /// No description provided for @playerAiYes.
  ///
  /// In ko, this message translates to:
  /// **'AI 생성 음원'**
  String get playerAiYes;

  /// No description provided for @playerAiNo.
  ///
  /// In ko, this message translates to:
  /// **'일반 음원'**
  String get playerAiNo;

  /// No description provided for @playerProviderLabel.
  ///
  /// In ko, this message translates to:
  /// **'AI 제공자'**
  String get playerProviderLabel;

  /// No description provided for @playerDisclosureLabel.
  ///
  /// In ko, this message translates to:
  /// **'고지 문구'**
  String get playerDisclosureLabel;

  /// No description provided for @playerNoDisclosure.
  ///
  /// In ko, this message translates to:
  /// **'별도 고지 없음'**
  String get playerNoDisclosure;

  /// No description provided for @playerBitrateLabel.
  ///
  /// In ko, this message translates to:
  /// **'비트레이트'**
  String get playerBitrateLabel;

  /// No description provided for @playerKeyLabel.
  ///
  /// In ko, this message translates to:
  /// **'키'**
  String get playerKeyLabel;

  /// No description provided for @playerPlayCountLabel.
  ///
  /// In ko, this message translates to:
  /// **'재생 수'**
  String get playerPlayCountLabel;

  /// No description provided for @playerStatusLabel.
  ///
  /// In ko, this message translates to:
  /// **'상태'**
  String get playerStatusLabel;

  /// No description provided for @playerAiNotice.
  ///
  /// In ko, this message translates to:
  /// **'이 음원은 AI 생성물 표시 의무에 따라 제공자와 생성 여부가 공개됩니다. 상업적 이용 범위는 제공자 라이선스 정책과 플랫폼 이용약관(TOS)을 따릅니다.'**
  String get playerAiNotice;

  /// No description provided for @playerViewTerms.
  ///
  /// In ko, this message translates to:
  /// **'이용약관 보기'**
  String get playerViewTerms;

  /// No description provided for @legalTitle.
  ///
  /// In ko, this message translates to:
  /// **'약관 및 안내'**
  String get legalTitle;

  /// No description provided for @legalAiTitle.
  ///
  /// In ko, this message translates to:
  /// **'AI 생성물 안내'**
  String get legalAiTitle;

  /// No description provided for @legalAiDesc.
  ///
  /// In ko, this message translates to:
  /// **'AI 음원 표시 및 라이선스 고지'**
  String get legalAiDesc;

  /// No description provided for @legalBlocksTitle.
  ///
  /// In ko, this message translates to:
  /// **'내 차단 내역'**
  String get legalBlocksTitle;

  /// No description provided for @legalBlocksDesc.
  ///
  /// In ko, this message translates to:
  /// **'신고 스트라이크에 따른 이용 제한'**
  String get legalBlocksDesc;

  /// No description provided for @legalWithdrawalTitle.
  ///
  /// In ko, this message translates to:
  /// **'회원 탈퇴 안내'**
  String get legalWithdrawalTitle;

  /// No description provided for @legalTermsHeader.
  ///
  /// In ko, this message translates to:
  /// **'이용약관'**
  String get legalTermsHeader;

  /// No description provided for @legalRequired.
  ///
  /// In ko, this message translates to:
  /// **'필수'**
  String get legalRequired;

  /// No description provided for @legalVersion.
  ///
  /// In ko, this message translates to:
  /// **'버전 {version}'**
  String legalVersion(String version);

  /// No description provided for @legalConsentRequired.
  ///
  /// In ko, this message translates to:
  /// **' · 필수 동의'**
  String get legalConsentRequired;

  /// No description provided for @legalConsent.
  ///
  /// In ko, this message translates to:
  /// **'동의하기'**
  String get legalConsent;

  /// No description provided for @legalConsentDone.
  ///
  /// In ko, this message translates to:
  /// **'동의가 기록되었습니다.'**
  String get legalConsentDone;

  /// No description provided for @legalConsentFailed.
  ///
  /// In ko, this message translates to:
  /// **'동의 처리에 실패했습니다.'**
  String get legalConsentFailed;

  /// No description provided for @legalLoadFailed.
  ///
  /// In ko, this message translates to:
  /// **'약관을 불러오지 못했습니다.'**
  String get legalLoadFailed;

  /// No description provided for @legalAiHeading.
  ///
  /// In ko, this message translates to:
  /// **'AI 음원 표시 안내'**
  String get legalAiHeading;

  /// No description provided for @legalAiP1.
  ///
  /// In ko, this message translates to:
  /// **'이 플랫폼의 모든 음원은 AI 생성 여부가 표시됩니다. 재생 화면의 \'AI·라이선스\' 탭에서 제공자(SUNO, UDIO, AIVA 등), 고지 문구, 비트레이트를 확인할 수 있습니다.'**
  String get legalAiP1;

  /// No description provided for @legalAiP2.
  ///
  /// In ko, this message translates to:
  /// **'AI 음원의 상업적 이용 범위는 각 제공자의 라이선스 정책과 플랫폼 이용약관을 따릅니다. 무단 보이스 클론, 무단 샘플 사용이 의심되는 곡은 재생 화면의 신고 메뉴로 제보할 수 있습니다.'**
  String get legalAiP2;

  /// No description provided for @legalViewTos.
  ///
  /// In ko, this message translates to:
  /// **'이용약관(TOS) 보기'**
  String get legalViewTos;

  /// No description provided for @legalNoBlocks.
  ///
  /// In ko, this message translates to:
  /// **'차단 내역이 없습니다.'**
  String get legalNoBlocks;

  /// No description provided for @legalStrikeItem.
  ///
  /// In ko, this message translates to:
  /// **'스트라이크 {number} · {reason}'**
  String legalStrikeItem(String number, String reason);

  /// No description provided for @legalNoExpiry.
  ///
  /// In ko, this message translates to:
  /// **'만료일 없음'**
  String get legalNoExpiry;

  /// No description provided for @legalExpires.
  ///
  /// In ko, this message translates to:
  /// **'만료: {date}'**
  String legalExpires(String date);

  /// No description provided for @legalBlocksFailed.
  ///
  /// In ko, this message translates to:
  /// **'차단 내역을 불러오지 못했습니다.'**
  String get legalBlocksFailed;

  /// No description provided for @legalWithdrawalHeading.
  ///
  /// In ko, this message translates to:
  /// **'탈퇴 전 확인하세요'**
  String get legalWithdrawalHeading;

  /// No description provided for @legalWithdrawalBody.
  ///
  /// In ko, this message translates to:
  /// **'· 플레이리스트와 재생 기록이 즉시 비활성화 및 삭제 처리됩니다.\n· 진행 중인 신고·분쟁이 있으면 관련 법령에 따라 보존 후 파기됩니다.\n· 아래 [즉시 회원 탈퇴] 버튼을 누르면 별도 고객센터 문의 없이 앱 내에서 즉시 탈퇴됩니다.'**
  String get legalWithdrawalBody;

  /// No description provided for @legalLogout.
  ///
  /// In ko, this message translates to:
  /// **'로그아웃'**
  String get legalLogout;

  /// No description provided for @legalWithdrawNow.
  ///
  /// In ko, this message translates to:
  /// **'즉시 회원 탈퇴'**
  String get legalWithdrawNow;

  /// No description provided for @storeTitle.
  ///
  /// In ko, this message translates to:
  /// **'매장 모드'**
  String get storeTitle;

  /// No description provided for @storeRefreshQueue.
  ///
  /// In ko, this message translates to:
  /// **'큐 새로고침'**
  String get storeRefreshQueue;

  /// No description provided for @storeUnpair.
  ///
  /// In ko, this message translates to:
  /// **'페어링 해제'**
  String get storeUnpair;

  /// No description provided for @storeUnpairTitle.
  ///
  /// In ko, this message translates to:
  /// **'페어링 해제'**
  String get storeUnpairTitle;

  /// No description provided for @storeUnpairBody.
  ///
  /// In ko, this message translates to:
  /// **'이 태블릿의 매장 페어링을 해제하고 재생을 중지합니다.'**
  String get storeUnpairBody;

  /// No description provided for @storeCancel.
  ///
  /// In ko, this message translates to:
  /// **'취소'**
  String get storeCancel;

  /// No description provided for @storeUnpairConfirm.
  ///
  /// In ko, this message translates to:
  /// **'해제'**
  String get storeUnpairConfirm;

  /// No description provided for @storePairGuide.
  ///
  /// In ko, this message translates to:
  /// **'파트너 콘솔의 매장 관리에서 발급된 8자리 페어링 코드를 입력하세요.'**
  String get storePairGuide;

  /// No description provided for @storePairCodeLabel.
  ///
  /// In ko, this message translates to:
  /// **'페어링 코드'**
  String get storePairCodeLabel;

  /// No description provided for @storePairCodeHint.
  ///
  /// In ko, this message translates to:
  /// **'예: A1B2C3D4'**
  String get storePairCodeHint;

  /// No description provided for @storePair.
  ///
  /// In ko, this message translates to:
  /// **'페어링'**
  String get storePair;

  /// No description provided for @storePairFailed.
  ///
  /// In ko, this message translates to:
  /// **'페어링에 실패했습니다.'**
  String get storePairFailed;

  /// No description provided for @storeOpenNow.
  ///
  /// In ko, this message translates to:
  /// **'{business} · 영업 중'**
  String storeOpenNow(String business);

  /// No description provided for @storeOpenNowPlaylist.
  ///
  /// In ko, this message translates to:
  /// **'{business} · 영업 중 · {playlist}'**
  String storeOpenNowPlaylist(String business, String playlist);

  /// No description provided for @storeClosed.
  ///
  /// In ko, this message translates to:
  /// **'{business} · {reason}'**
  String storeClosed(String business, String reason);

  /// No description provided for @storeQueueEmpty.
  ///
  /// In ko, this message translates to:
  /// **'재생할 곡이 없습니다.'**
  String get storeQueueEmpty;

  /// No description provided for @storeNoSelection.
  ///
  /// In ko, this message translates to:
  /// **'선택된 곡 없음'**
  String get storeNoSelection;

  /// No description provided for @storePrev.
  ///
  /// In ko, this message translates to:
  /// **'이전 곡'**
  String get storePrev;

  /// No description provided for @storePause.
  ///
  /// In ko, this message translates to:
  /// **'일시정지'**
  String get storePause;

  /// No description provided for @storePlay.
  ///
  /// In ko, this message translates to:
  /// **'재생'**
  String get storePlay;

  /// No description provided for @storeNext.
  ///
  /// In ko, this message translates to:
  /// **'다음 곡'**
  String get storeNext;

  /// No description provided for @errNotStreamable.
  ///
  /// In ko, this message translates to:
  /// **'재생할 수 없는 곡입니다 (차단 또는 심사 중). 다음 곡으로 이동합니다.'**
  String get errNotStreamable;

  /// No description provided for @errNotFound.
  ///
  /// In ko, this message translates to:
  /// **'요청한 항목을 찾을 수 없습니다.'**
  String get errNotFound;

  /// No description provided for @errAuth.
  ///
  /// In ko, this message translates to:
  /// **'로그인이 필요하거나 인증이 만료되었습니다.'**
  String get errAuth;

  /// No description provided for @errInactive.
  ///
  /// In ko, this message translates to:
  /// **'비활성화된 계정입니다. 고객센터에 문의하세요.'**
  String get errInactive;

  /// No description provided for @errForbidden.
  ///
  /// In ko, this message translates to:
  /// **'권한이 없습니다.'**
  String get errForbidden;

  /// No description provided for @errRateLimited.
  ///
  /// In ko, this message translates to:
  /// **'요청이 너무 많습니다. 잠시 후 다시 시도하세요.'**
  String get errRateLimited;

  /// No description provided for @errEmailTaken.
  ///
  /// In ko, this message translates to:
  /// **'이미 등록된 이메일입니다.'**
  String get errEmailTaken;

  /// No description provided for @errTotpRequired.
  ///
  /// In ko, this message translates to:
  /// **'2단계 인증 코드가 필요합니다.'**
  String get errTotpRequired;

  /// No description provided for @errAgeConfirm.
  ///
  /// In ko, this message translates to:
  /// **'만 14세 이상 확인이 필요합니다. 프로필에서 연령을 확인하세요.'**
  String get errAgeConfirm;

  /// No description provided for @errConsent.
  ///
  /// In ko, this message translates to:
  /// **'필수 약관 동의가 필요합니다. 약관 화면에서 동의하세요.'**
  String get errConsent;

  /// No description provided for @errTotpInvalid.
  ///
  /// In ko, this message translates to:
  /// **'인증 코드가 올바르지 않습니다. 다시 입력하세요.'**
  String get errTotpInvalid;

  /// No description provided for @errLoginLocked.
  ///
  /// In ko, this message translates to:
  /// **'로그인 시도가 너무 많습니다. 15분 후 다시 시도하세요.'**
  String get errLoginLocked;

  /// No description provided for @errProviderNotAllowed.
  ///
  /// In ko, this message translates to:
  /// **'허용 목록에 없는 제공자입니다. 허용 제휴 제공자를 선택하세요.'**
  String get errProviderNotAllowed;

  /// No description provided for @errInvalidInput.
  ///
  /// In ko, this message translates to:
  /// **'입력값이 올바르지 않습니다: {detail}'**
  String errInvalidInput(String detail);

  /// No description provided for @errGeneric.
  ///
  /// In ko, this message translates to:
  /// **'일시적인 오류가 발생했습니다.'**
  String get errGeneric;

  /// No description provided for @errNetwork.
  ///
  /// In ko, this message translates to:
  /// **'네트워크 연결에 실패했습니다.'**
  String get errNetwork;

  /// No description provided for @errRequest.
  ///
  /// In ko, this message translates to:
  /// **'요청에 실패했습니다.'**
  String get errRequest;

  /// No description provided for @reasonAiPolicy.
  ///
  /// In ko, this message translates to:
  /// **'AI 정책 위반'**
  String get reasonAiPolicy;

  /// No description provided for @reasonVoiceClone.
  ///
  /// In ko, this message translates to:
  /// **'무단 보이스 클론'**
  String get reasonVoiceClone;

  /// No description provided for @reasonSample.
  ///
  /// In ko, this message translates to:
  /// **'무단 샘플 사용'**
  String get reasonSample;

  /// No description provided for @reasonOther.
  ///
  /// In ko, this message translates to:
  /// **'기타'**
  String get reasonOther;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'ja', 'ko', 'th', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'ja':
      return AppLocalizationsJa();
    case 'ko':
      return AppLocalizationsKo();
    case 'th':
      return AppLocalizationsTh();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
      'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
      'an issue with the localizations generation tool. Please file an issue '
      'on GitHub with a reproducible sample app and the gen-l10n configuration '
      'that was used.');
}
