// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'AVIBE';

  @override
  String get tabHome => 'Home';

  @override
  String get tabSearch => 'Search';

  @override
  String get tabPlaylists => 'Lists';

  @override
  String get tabPlayer => 'Player';

  @override
  String get tabStore => 'Store';

  @override
  String get sessionExpired => 'Your session expired. Please sign in again.';

  @override
  String get loginTitle => 'Sign in to AVIBE';

  @override
  String get loginEmail => 'Email';

  @override
  String get loginPassword => 'Password (8+ characters)';

  @override
  String get loginTotp => 'Two-factor code';

  @override
  String get loginTotpHint => '6-digit app code or recovery code';

  @override
  String get loginSubmit => 'Sign in';

  @override
  String get loginSubmitTotp => 'Verify and sign in';

  @override
  String get loginCreateAccount => 'Create account';

  @override
  String get loginHaveAccount => 'I already have an account';

  @override
  String get loginNeedAgeConfirm => 'You must confirm you are 14 or older.';

  @override
  String get loginTotpPrompt =>
      'Enter the 6-digit code from your authenticator app.';

  @override
  String get loginFailed => 'Sign-in failed.';

  @override
  String get loginOrSocial => 'Or continue with a social account';

  @override
  String loginSocialFailed(String provider) {
    return '$provider sign-in failed.';
  }

  @override
  String get loginSocialCancelled => 'Social sign-in was cancelled.';

  @override
  String get loginSocialReturnFailed =>
      'Failed to return to the app. Please try again.';

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
  String get signupTitle => 'Create your AVIBE account';

  @override
  String get signupDisplayName => 'Display name';

  @override
  String get signupBirthYear => 'Birth year (optional)';

  @override
  String get signupBirthYearHint => 'e.g. 1995';

  @override
  String get signupInviteCode => 'Invite code (if required)';

  @override
  String get signupAgeConfirm =>
      'I am 14 or older. (Required; under 14 cannot sign up)';

  @override
  String get signupSubmit => 'Sign up';

  @override
  String get signupBackToLogin => 'Back to sign-in';

  @override
  String get ageGateTitle => 'Confirm you are 14+';

  @override
  String get ageGateDesc =>
      'You signed in with a social account for the first time. Age confirmation is required to use streaming and payments.';

  @override
  String get ageGateConfirm => 'I am 14 or older. (Required)';

  @override
  String get ageGateBirthYear => 'Birth year (optional)';

  @override
  String get ageGateBirthYearHint => 'e.g. 1995';

  @override
  String get ageGateFailed => 'Age confirmation failed.';

  @override
  String get ageGateLogout => 'Sign out';

  @override
  String get ageGateStart => 'Confirm and start';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get settingsLanguage => 'Language';

  @override
  String get settingsLanguageSaved => 'Language saved.';

  @override
  String get settingsLanguageFailed => 'Failed to save the language.';

  @override
  String get settingsServerNote =>
      'When signed in, this syncs with your server language setting.';

  @override
  String get settingsLogout => 'Sign out';

  @override
  String get settingsLegal => 'Terms & notices';

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
  String get homeEmpty => 'No tracks published yet.';

  @override
  String get homeFailed => 'Couldn\'t load the feed.';

  @override
  String get searchTitle => 'Search';

  @override
  String get searchHint => 'Search by track title or AI provider';

  @override
  String get searchEmpty => 'No results found.';

  @override
  String get searchFailed => 'Search failed.';

  @override
  String get playlistTitle => 'Playlists';

  @override
  String get playlistEmpty => 'No playlists yet. Tap + to create one.';

  @override
  String get playlistPublic => 'Public';

  @override
  String get playlistPrivate => 'Private';

  @override
  String get playlistRename => 'Rename';

  @override
  String get playlistDelete => 'Delete';

  @override
  String get playlistListFailed => 'Couldn\'t load the list.';

  @override
  String get playlistNew => 'New playlist';

  @override
  String get playlistEdit => 'Edit playlist';

  @override
  String get playlistNameLabel => 'Title';

  @override
  String get playlistCancel => 'Cancel';

  @override
  String get playlistSave => 'Save';

  @override
  String get playlistSaveFailed => 'Save failed.';

  @override
  String get playlistDeleteFailed => 'Delete failed.';

  @override
  String get playlistSaveOrder => 'Save order';

  @override
  String get playlistAddTrack => 'Add track';

  @override
  String playlistDetailEmpty(String title) {
    return '\'$title\' has no tracks.\nTap + to add some.';
  }

  @override
  String get playlistPlay => 'Play';

  @override
  String get playlistDetailFailed => 'Couldn\'t load details.';

  @override
  String get playlistOrderFailed => 'Couldn\'t save the order.';

  @override
  String get playlistClose => 'Close';

  @override
  String get playlistAddFailed => 'Couldn\'t add the track.';

  @override
  String get playerTitle => 'Now playing';

  @override
  String get playerReportBlock => 'Report / block';

  @override
  String get playerTabWaveform => 'Waveform';

  @override
  String get playerTabLyrics => 'Lyrics';

  @override
  String get playerTabAi => 'AI & license';

  @override
  String get playerNoTrack => 'Nothing is playing.';

  @override
  String playerError(String detail) {
    return 'Player error: $detail';
  }

  @override
  String get playerReportReasonTitle => 'Choose a report reason';

  @override
  String get playerBlockArtist =>
      'Block this artist (partner) and remove from queue';

  @override
  String get playerViewBlocks => 'View my block list';

  @override
  String get playerBlockReasonDefault =>
      'Direct artist block from the mobile app user';

  @override
  String get playerArtistBlocked =>
      'The artist was blocked and removed from playback immediately.';

  @override
  String get playerBlockFailed => 'Couldn\'t block the artist.';

  @override
  String get playerReportTitle => 'Report';

  @override
  String get playerReportDetailLabel => 'Details (optional)';

  @override
  String get playerReportDetailHint => 'Describe the rights issue';

  @override
  String get playerCancel => 'Cancel';

  @override
  String get playerReport => 'Report';

  @override
  String playerReportReceived(String status) {
    return 'Report received ($status).';
  }

  @override
  String get playerReportFailed => 'Couldn\'t submit the report.';

  @override
  String get playerWaveformTitle => 'Audio waveform';

  @override
  String get playerInstrumental => 'This is an instrumental track.';

  @override
  String playerNoLyrics(String provider) {
    return 'No lyrics registered.\nThis track was generated with the $provider AI model.';
  }

  @override
  String playerPromptLabel(String prompt) {
    return 'Generation prompt: $prompt';
  }

  @override
  String get playerTrackFailed => 'Couldn\'t load track info.';

  @override
  String get playerAiLabel => 'AI-generated';

  @override
  String get playerAiYes => 'AI-generated track';

  @override
  String get playerAiNo => 'Regular track';

  @override
  String get playerProviderLabel => 'AI provider';

  @override
  String get playerDisclosureLabel => 'Disclosure';

  @override
  String get playerNoDisclosure => 'No separate disclosure';

  @override
  String get playerBitrateLabel => 'Bitrate';

  @override
  String get playerKeyLabel => 'Key';

  @override
  String get playerPlayCountLabel => 'Plays';

  @override
  String get playerStatusLabel => 'Status';

  @override
  String get playerAiNotice =>
      'This track\'s provider and AI status are disclosed under the AI-generated content labeling duty. Commercial use follows each provider\'s license policy and the platform Terms of Service (TOS).';

  @override
  String get playerViewTerms => 'View terms';

  @override
  String get legalTitle => 'Terms & notices';

  @override
  String get legalAiTitle => 'AI content notice';

  @override
  String get legalAiDesc => 'AI track labeling and license notices';

  @override
  String get legalBlocksTitle => 'My block list';

  @override
  String get legalBlocksDesc => 'Restrictions from report strikes';

  @override
  String get legalWithdrawalTitle => 'Account withdrawal';

  @override
  String get legalTermsHeader => 'Terms of service';

  @override
  String get legalRequired => 'Required';

  @override
  String legalVersion(String version) {
    return 'Version $version';
  }

  @override
  String get legalConsentRequired => ' · consent required';

  @override
  String get legalConsent => 'Agree';

  @override
  String get legalConsentDone => 'Your consent was recorded.';

  @override
  String get legalConsentFailed => 'Couldn\'t record consent.';

  @override
  String get legalLoadFailed => 'Couldn\'t load the terms.';

  @override
  String get legalAiHeading => 'AI track labeling';

  @override
  String get legalAiP1 =>
      'Every track on this platform is labeled for AI generation. Check the provider (SUNO, UDIO, AIVA, etc.), disclosure text, and bitrate in the player screen\'s \'AI & license\' tab.';

  @override
  String get legalAiP2 =>
      'Commercial use of AI tracks follows each provider\'s license policy and the platform Terms. Report suspected unauthorized voice clones or samples from the player screen\'s report menu.';

  @override
  String get legalViewTos => 'View Terms of Service (TOS)';

  @override
  String get legalNoBlocks => 'No blocks on record.';

  @override
  String legalStrikeItem(String number, String reason) {
    return 'Strike $number · $reason';
  }

  @override
  String get legalNoExpiry => 'No expiry';

  @override
  String legalExpires(String date) {
    return 'Expires: $date';
  }

  @override
  String get legalBlocksFailed => 'Couldn\'t load the block list.';

  @override
  String get legalWithdrawalHeading => 'Before you withdraw';

  @override
  String get legalWithdrawalBody =>
      '· Playlists and listening history are deactivated and deleted immediately.\n· Ongoing reports or disputes are retained under applicable laws, then destroyed.\n· Tapping [Withdraw now] below withdraws immediately in the app, no support ticket needed.';

  @override
  String get legalLogout => 'Sign out';

  @override
  String get legalWithdrawNow => 'Withdraw now';

  @override
  String get storeTitle => 'Store mode';

  @override
  String get storeRefreshQueue => 'Refresh queue';

  @override
  String get storeUnpair => 'Unpair';

  @override
  String get storeUnpairTitle => 'Unpair';

  @override
  String get storeUnpairBody =>
      'Unpair this tablet from the store and stop playback.';

  @override
  String get storeCancel => 'Cancel';

  @override
  String get storeUnpairConfirm => 'Unpair';

  @override
  String get storePairGuide =>
      'Enter the 8-character pairing code issued in the partner console\'s store management.';

  @override
  String get storePairCodeLabel => 'Pairing code';

  @override
  String get storePairCodeHint => 'e.g. A1B2C3D4';

  @override
  String get storePair => 'Pair';

  @override
  String get storePairFailed => 'Pairing failed.';

  @override
  String storeOpenNow(String business) {
    return '$business · Open now';
  }

  @override
  String storeOpenNowPlaylist(String business, String playlist) {
    return '$business · Open now · $playlist';
  }

  @override
  String storeClosed(String business, String reason) {
    return '$business · $reason';
  }

  @override
  String get storeQueueEmpty => 'No tracks to play.';

  @override
  String get storeNoSelection => 'No track selected';

  @override
  String get storePrev => 'Previous track';

  @override
  String get storePause => 'Pause';

  @override
  String get storePlay => 'Play';

  @override
  String get storeNext => 'Next track';

  @override
  String get errNotStreamable =>
      'This track can\'t be played (blocked or in review). Moving to the next track.';

  @override
  String get errNotFound => 'The requested item was not found.';

  @override
  String get errAuth => 'Sign-in is required or your session expired.';

  @override
  String get errInactive => 'This account is deactivated. Contact support.';

  @override
  String get errForbidden => 'You don\'t have permission.';

  @override
  String get errRateLimited => 'Too many requests. Please try again shortly.';

  @override
  String get errEmailTaken => 'This email is already registered.';

  @override
  String get errTotpRequired => 'A two-factor code is required.';

  @override
  String get errAgeConfirm =>
      '14+ confirmation is required. Confirm your age in your profile.';

  @override
  String get errConsent =>
      'Required terms consent is needed. Agree on the terms screen.';

  @override
  String get errTotpInvalid => 'The code is incorrect. Please try again.';

  @override
  String get errLoginLocked =>
      'Too many sign-in attempts. Try again in 15 minutes.';

  @override
  String get errProviderNotAllowed =>
      'This provider is not allowlisted. Choose an allowed partner provider.';

  @override
  String errInvalidInput(String detail) {
    return 'Invalid input: $detail';
  }

  @override
  String get errGeneric => 'A temporary error occurred.';

  @override
  String get errNetwork => 'Couldn\'t connect to the network.';

  @override
  String get errRequest => 'The request failed.';

  @override
  String get reasonAiPolicy => 'AI policy violation';

  @override
  String get reasonVoiceClone => 'Unauthorized voice clone';

  @override
  String get reasonSample => 'Unauthorized sample use';

  @override
  String get reasonOther => 'Other';

  @override
  String get certTitle => 'Certificate';

  @override
  String get certNumber => 'Certificate number';

  @override
  String get certIssuedAt => 'Issued at';

  @override
  String get certPlanTier => 'Plan tier';

  @override
  String get certGates => 'Verification gates';

  @override
  String get certGatesPassed => 'Passed';

  @override
  String get certGatesFailed => 'Not passed';

  @override
  String get certNotFound => 'Certificate not found.';

  @override
  String get certLoadFailed => 'Failed to load the certificate.';

  @override
  String get playerCertificateLabel => 'Certificate';

  @override
  String get completeEmailTitle => 'Enter your email';

  @override
  String get completeEmailSubtitle =>
      'Your social account did not share an email, so we need one more step.';

  @override
  String get completeEmailHint => 'you@example.com';

  @override
  String get completeEmailSubmit => 'Continue';

  @override
  String get completeEmailBackToLogin => 'Back to login';

  @override
  String get completeEmailInvalid => 'Enter a valid email address.';

  @override
  String get completeEmailFailed => 'Submission failed.';

  @override
  String get completeEmailPendingTitle => 'Verification email sent';

  @override
  String completeEmailPendingBody(String email) {
    return 'We sent a verification email to $email. Open the link in the mail to finish signing up.';
  }
}
