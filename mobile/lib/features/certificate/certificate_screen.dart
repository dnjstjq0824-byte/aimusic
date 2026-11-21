// Public certificate verification (GET /certificates/:number).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_locale.dart';
import '../../core/error.dart';
import '../../core/models.dart';
import '../../core/providers.dart';

class CertificateScreen extends ConsumerWidget {
  const CertificateScreen({super.key, required this.number});

  final String number;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = context.l10n;
    final viewAsync = ref.watch(_certificateProvider(number));
    return Scaffold(
      appBar: AppBar(title: Text(s.certTitle)),
      body: viewAsync.when(
        data: (view) {
          if (view == null) {
            return Center(child: Text(s.certNotFound));
          }
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Row(
                children: [
                  Icon(
                    view.gatesPassed
                        ? Icons.verified
                        : Icons.verified_outlined,
                    color: Theme.of(context).colorScheme.primary,
                    size: 40,
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          view.title,
                          style: const TextStyle(
                              fontSize: 18, fontWeight: FontWeight.w600),
                        ),
                        Text(
                          view.certificate.certNumber,
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              _row(context, s.certNumber, view.certificate.certNumber),
              _row(context, s.certIssuedAt, view.certificate.issuedAt),
              _row(context, s.playerProviderLabel, view.aiProvider),
              _row(context, s.playerStatusLabel, view.status),
              if (view.isrc != null) _row(context, 'ISRC', view.isrc!),
              if (view.planTier != null)
                _row(context, s.certPlanTier, view.planTier!),
              _row(
                context,
                s.certGates,
                view.gatesPassed ? s.certGatesPassed : s.certGatesFailed,
              ),
            ],
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Text(e is ApiException
              ? friendlyMessage(e, s)
              : s.certLoadFailed),
        ),
      ),
    );
  }

  Widget _row(BuildContext context, String label, String value) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 120,
              child: Text(label,
                  style: Theme.of(context).textTheme.bodySmall),
            ),
            Expanded(
              child: Text(value,
                  style: const TextStyle(fontWeight: FontWeight.w600)),
            ),
          ],
        ),
      );
}

final _certificateProvider =
    FutureProvider.family<CertificateView?, String>((ref, number) async {
  return ref.watch(apiProvider).verifyCertificate(number);
});
