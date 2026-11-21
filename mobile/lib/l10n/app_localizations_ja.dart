// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Japanese (`ja`).
class AppLocalizationsJa extends AppLocalizations {
  AppLocalizationsJa([String locale = 'ja']) : super(locale);

  @override
  String get appTitle => 'AVIBE';

  @override
  String get tabHome => 'ホーム';

  @override
  String get tabSearch => '検索';

  @override
  String get tabPlaylists => 'リスト';

  @override
  String get tabPlayer => '再生';

  @override
  String get tabStore => '店舗';

  @override
  String get sessionExpired => 'セッションが期限切れです。もう一度ログインしてください。';

  @override
  String get loginTitle => 'AVIBEにログイン';

  @override
  String get loginEmail => 'メールアドレス';

  @override
  String get loginPassword => 'パスワード（8文字以上）';

  @override
  String get loginTotp => '2段階認証コード';

  @override
  String get loginTotpHint => '認証アプリの6桁コードまたは復旧コード';

  @override
  String get loginSubmit => 'ログイン';

  @override
  String get loginSubmitTotp => '認証してログイン';

  @override
  String get loginCreateAccount => 'アカウント作成';

  @override
  String get loginHaveAccount => 'すでにアカウントをお持ちの方';

  @override
  String get loginNeedAgeConfirm => '14歳以上であることの確認が必要です。';

  @override
  String get loginTotpPrompt => '認証アプリの6桁コードを入力してください。';

  @override
  String get loginFailed => '認証に失敗しました。';

  @override
  String get loginOrSocial => 'またはソーシャルアカウントで続ける';

  @override
  String loginSocialFailed(String provider) {
    return '$providerでのログインに失敗しました。';
  }

  @override
  String get loginSocialCancelled => 'ソーシャルログインがキャンセルされました。';

  @override
  String get loginSocialReturnFailed => 'アプリへの復帰に失敗しました。もう一度お試しください。';

  @override
  String get providerGoogle => 'Google';

  @override
  String get providerKakao => 'カカオ';

  @override
  String get providerApple => 'Apple';

  @override
  String get providerNaver => 'ネイバー';

  @override
  String get providerLine => 'LINE';

  @override
  String get signupTitle => 'AVIBEアカウント作成';

  @override
  String get signupDisplayName => '表示名';

  @override
  String get signupBirthYear => '生年（任意）';

  @override
  String get signupBirthYearHint => '例：1995';

  @override
  String get signupInviteCode => '招待コード（必要な場合）';

  @override
  String get signupAgeConfirm => '14歳以上です。（必須、14歳未満は登録不可）';

  @override
  String get signupSubmit => '新規登録';

  @override
  String get signupBackToLogin => 'ログインに戻る';

  @override
  String get ageGateTitle => '14歳以上の確認';

  @override
  String get ageGateDesc => 'ソーシャルアカウントで初めてログインしました。ストリーミング・決済の利用には年齢確認が必要です。';

  @override
  String get ageGateConfirm => '14歳以上です。（必須）';

  @override
  String get ageGateBirthYear => '生年（任意）';

  @override
  String get ageGateBirthYearHint => '例：1995';

  @override
  String get ageGateFailed => '年齢確認に失敗しました。';

  @override
  String get ageGateLogout => 'ログアウト';

  @override
  String get ageGateStart => '確認して開始';

  @override
  String get settingsTitle => '設定';

  @override
  String get settingsLanguage => '言語';

  @override
  String get settingsLanguageSaved => '言語を保存しました。';

  @override
  String get settingsLanguageFailed => '言語の保存に失敗しました。';

  @override
  String get settingsServerNote => 'ログインするとサーバーの言語設定と同期されます。';

  @override
  String get settingsLogout => 'ログアウト';

  @override
  String get settingsLegal => '利用規約・お知らせ';

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
  String get homeEmpty => 'まだ公開された曲がありません。';

  @override
  String get homeFailed => 'フィードを読み込めませんでした。';

  @override
  String get searchTitle => '検索';

  @override
  String get searchHint => '曲名またはAIプロバイダーで検索';

  @override
  String get searchEmpty => '検索結果がありません。';

  @override
  String get searchFailed => '検索に失敗しました。';

  @override
  String get playlistTitle => 'プレイリスト';

  @override
  String get playlistEmpty => 'プレイリストがありません。+ボタンで作成してください。';

  @override
  String get playlistPublic => '公開';

  @override
  String get playlistPrivate => '非公開';

  @override
  String get playlistRename => '名前の変更';

  @override
  String get playlistDelete => '削除';

  @override
  String get playlistListFailed => '一覧を読み込めませんでした。';

  @override
  String get playlistNew => '新しいプレイリスト';

  @override
  String get playlistEdit => 'プレイリストを編集';

  @override
  String get playlistNameLabel => 'タイトル';

  @override
  String get playlistCancel => 'キャンセル';

  @override
  String get playlistSave => '保存';

  @override
  String get playlistSaveFailed => '保存に失敗しました。';

  @override
  String get playlistDeleteFailed => '削除に失敗しました。';

  @override
  String get playlistSaveOrder => '順序を保存';

  @override
  String get playlistAddTrack => '曲を追加';

  @override
  String playlistDetailEmpty(String title) {
    return '「$title」に曲がありません。\n+ボタンで追加してください。';
  }

  @override
  String get playlistPlay => '再生';

  @override
  String get playlistDetailFailed => '詳細を読み込めませんでした。';

  @override
  String get playlistOrderFailed => '順序の保存に失敗しました。';

  @override
  String get playlistClose => '閉じる';

  @override
  String get playlistAddFailed => '追加に失敗しました。';

  @override
  String get playerTitle => '再生中';

  @override
  String get playerReportBlock => '通報 / ブロック';

  @override
  String get playerTabWaveform => '波形';

  @override
  String get playerTabLyrics => '歌詞';

  @override
  String get playerTabAi => 'AI・ライセンス';

  @override
  String get playerNoTrack => '再生中の曲がありません。';

  @override
  String playerError(String detail) {
    return 'プレーヤーエラー：$detail';
  }

  @override
  String get playerReportReasonTitle => '通報理由の選択';

  @override
  String get playerBlockArtist => 'このアーティスト（パートナー）をブロックしてキューから除外';

  @override
  String get playerViewBlocks => 'ブロック履歴を見る';

  @override
  String get playerBlockReasonDefault => 'モバイルアプリユーザーによる直接のアーティストブロック';

  @override
  String get playerArtistBlocked => 'アーティストがブロックされ、再生リストから即時除外されました。';

  @override
  String get playerBlockFailed => 'アーティストのブロックに失敗しました。';

  @override
  String get playerReportTitle => '通報する';

  @override
  String get playerReportDetailLabel => '詳細（任意）';

  @override
  String get playerReportDetailHint => '権利侵害の内容を説明してください';

  @override
  String get playerCancel => 'キャンセル';

  @override
  String get playerReport => '通報';

  @override
  String playerReportReceived(String status) {
    return '通報を受け付けました（$status）。';
  }

  @override
  String get playerReportFailed => '通報に失敗しました。';

  @override
  String get playerWaveformTitle => 'オーディオ波形 (Waveform)';

  @override
  String get playerInstrumental => 'インストゥルメンタル曲です。';

  @override
  String playerNoLyrics(String provider) {
    return '登録された歌詞がありません。\nこの曲は$provider AIモデルで生成された音源です。';
  }

  @override
  String playerPromptLabel(String prompt) {
    return '生成プロンプト：$prompt';
  }

  @override
  String get playerTrackFailed => '曲情報を読み込めませんでした。';

  @override
  String get playerAiLabel => 'AI生成の有無';

  @override
  String get playerAiYes => 'AI生成音源';

  @override
  String get playerAiNo => '通常音源';

  @override
  String get playerProviderLabel => 'AIプロバイダー';

  @override
  String get playerDisclosureLabel => '告知文';

  @override
  String get playerNoDisclosure => '別途告知なし';

  @override
  String get playerBitrateLabel => 'ビットレート';

  @override
  String get playerKeyLabel => 'キー';

  @override
  String get playerPlayCountLabel => '再生数';

  @override
  String get playerStatusLabel => 'ステータス';

  @override
  String get playerAiNotice =>
      'この音源はAI生成物の表示義務に従い、プロバイダーと生成有無が公開されます。商用利用範囲は各プロバイダーのライセンス方針とプラットフォーム利用規約（TOS）に従います。';

  @override
  String get playerViewTerms => '利用規約を見る';

  @override
  String get legalTitle => '利用規約・お知らせ';

  @override
  String get legalAiTitle => 'AI生成物のお知らせ';

  @override
  String get legalAiDesc => 'AI音源の表示とライセンス告知';

  @override
  String get legalBlocksTitle => 'ブロック履歴';

  @override
  String get legalBlocksDesc => '通報ストライクによる利用制限';

  @override
  String get legalWithdrawalTitle => '退会のご案内';

  @override
  String get legalTermsHeader => '利用規約';

  @override
  String get legalRequired => '必須';

  @override
  String legalVersion(String version) {
    return 'バージョン $version';
  }

  @override
  String get legalConsentRequired => ' · 同意必須';

  @override
  String get legalConsent => '同意する';

  @override
  String get legalConsentDone => '同意が記録されました。';

  @override
  String get legalConsentFailed => '同意の処理に失敗しました。';

  @override
  String get legalLoadFailed => '規約を読み込めませんでした。';

  @override
  String get legalAiHeading => 'AI音源の表示について';

  @override
  String get legalAiP1 =>
      'このプラットフォームのすべての音源にはAI生成有無が表示されます。再生画面の「AI・ライセンス」タブでプロバイダー（SUNO、UDIO、AIVAなど）、告知文、ビットレートを確認できます。';

  @override
  String get legalAiP2 =>
      'AI音源の商用利用範囲は各プロバイダーのライセンス方針とプラットフォーム利用規約に従います。無断ボイスクローンや無断サンプル使用が疑われる曲は、再生画面の通報メニューから報告できます。';

  @override
  String get legalViewTos => '利用規約（TOS）を見る';

  @override
  String get legalNoBlocks => 'ブロック履歴がありません。';

  @override
  String legalStrikeItem(String number, String reason) {
    return 'ストライク $number · $reason';
  }

  @override
  String get legalNoExpiry => '有効期限なし';

  @override
  String legalExpires(String date) {
    return '期限：$date';
  }

  @override
  String get legalBlocksFailed => 'ブロック履歴を読み込めませんでした。';

  @override
  String get legalWithdrawalHeading => '退会前にご確認ください';

  @override
  String get legalWithdrawalBody =>
      '・プレイリストと再生履歴は即時無効化・削除されます。\n・進行中の通報・紛争がある場合は関連法令に従い保存後に破棄されます。\n・下の［今すぐ退会］ボタンを押すと、カスタマーセンターへの問い合わせなしにアプリ内ですぐに退会されます。';

  @override
  String get legalLogout => 'ログアウト';

  @override
  String get legalWithdrawNow => '今すぐ退会';

  @override
  String get storeTitle => '店舗モード';

  @override
  String get storeRefreshQueue => 'キューを更新';

  @override
  String get storeUnpair => 'ペアリング解除';

  @override
  String get storeUnpairTitle => 'ペアリング解除';

  @override
  String get storeUnpairBody => 'このタブレットの店舗ペアリングを解除して再生を停止します。';

  @override
  String get storeCancel => 'キャンセル';

  @override
  String get storeUnpairConfirm => '解除';

  @override
  String get storePairGuide => 'パートナーコンソールの店舗管理で発行された8文字のペアリングコードを入力してください。';

  @override
  String get storePairCodeLabel => 'ペアリングコード';

  @override
  String get storePairCodeHint => '例：A1B2C3D4';

  @override
  String get storePair => 'ペアリング';

  @override
  String get storePairFailed => 'ペアリングに失敗しました。';

  @override
  String storeOpenNow(String business) {
    return '$business · 営業中';
  }

  @override
  String storeOpenNowPlaylist(String business, String playlist) {
    return '$business · 営業中 · $playlist';
  }

  @override
  String storeClosed(String business, String reason) {
    return '$business · $reason';
  }

  @override
  String get storeQueueEmpty => '再生する曲がありません。';

  @override
  String get storeNoSelection => '選択中の曲なし';

  @override
  String get storePrev => '前の曲';

  @override
  String get storePause => '一時停止';

  @override
  String get storePlay => '再生';

  @override
  String get storeNext => '次の曲';

  @override
  String get errNotStreamable => '再生できない曲です（ブロックまたは審査中）。次の曲に移動します。';

  @override
  String get errNotFound => 'リクエストした項目が見つかりません。';

  @override
  String get errAuth => 'ログインが必要か、認証が期限切れです。';

  @override
  String get errInactive => '無効化されたアカウントです。カスタマーセンターにお問い合わせください。';

  @override
  String get errForbidden => '権限がありません。';

  @override
  String get errRateLimited => 'リクエストが多すぎます。しばらくしてからお試しください。';

  @override
  String get errEmailTaken => 'このメールアドレスは既に登録されています。';

  @override
  String get errTotpRequired => '2段階認証コードが必要です。';

  @override
  String get errAgeConfirm => '14歳以上の確認が必要です。プロフィールで年齢を確認してください。';

  @override
  String get errConsent => '必須規約への同意が必要です。規約画面で同意してください。';

  @override
  String get errTotpInvalid => '認証コードが正しくありません。もう一度入力してください。';

  @override
  String get errLoginLocked => 'ログイン試行が多すぎます。15分後にもう一度お試しください。';

  @override
  String get errProviderNotAllowed =>
      '許可リストにないプロバイダーです。許可された提携プロバイダーを選択してください。';

  @override
  String errInvalidInput(String detail) {
    return '入力値が正しくありません：$detail';
  }

  @override
  String get errGeneric => '一時的なエラーが発生しました。';

  @override
  String get errNetwork => 'ネットワークに接続できませんでした。';

  @override
  String get errRequest => 'リクエストに失敗しました。';

  @override
  String get reasonAiPolicy => 'AIポリシー違反';

  @override
  String get reasonVoiceClone => '無断ボイスクローン';

  @override
  String get reasonSample => '無断サンプル使用';

  @override
  String get reasonOther => 'その他';

  @override
  String get certTitle => '証明書の確認';

  @override
  String get certNumber => '証明書番号';

  @override
  String get certIssuedAt => '発行日';

  @override
  String get certPlanTier => 'プラン';

  @override
  String get certGates => '検証ゲート';

  @override
  String get certGatesPassed => '合格';

  @override
  String get certGatesFailed => '不合格';

  @override
  String get certNotFound => '証明書が見つかりません。';

  @override
  String get certLoadFailed => '証明書を読み込めませんでした。';

  @override
  String get playerCertificateLabel => '証明書';

  @override
  String get completeEmailTitle => 'メールの入力';

  @override
  String get completeEmailSubtitle => 'ソーシャルアカウントからメールが提供されなかったため、追加の入力が必要です。';

  @override
  String get completeEmailHint => 'you@example.com';

  @override
  String get completeEmailSubmit => '続ける';

  @override
  String get completeEmailBackToLogin => 'ログインに戻る';

  @override
  String get completeEmailInvalid => '有効なメールアドレスを入力してください。';

  @override
  String get completeEmailFailed => '送信に失敗しました。';

  @override
  String get completeEmailPendingTitle => '認証メールを送信しました';

  @override
  String completeEmailPendingBody(String email) {
    return '$email に認証メールを送信しました。メール内のリンクを開くと登録が完了します。';
  }
}
