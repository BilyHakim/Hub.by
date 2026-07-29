-- +goose Up

ALTER TABLE pyramid_items
ADD CONSTRAINT pyramid_items_workspace_priority_title_key UNIQUE(workspace_id, priority, title);

INSERT INTO pyramid_items(workspace_id,priority,title,is_completed,tracker_module)
SELECT w.id, defaults.priority, defaults.title, FALSE, defaults.tracker
FROM workspaces w
CROSS JOIN (VALUES
    (1,'Punya penghasilan tetap dalam 3 bulan terakhir','cashflow'),
    (1,'Punya penghasilan tetap dalam 6 bulan terakhir','cashflow'),
    (1,'Punya penghasilan tetap dalam 12 bulan terakhir','cashflow'),
    (1,'Punya minimal 1 sumber penghasilan tambahan','cashflow'),
    (2,'Pemasukan lebih besar dari pengeluaran dalam 3 bulan terakhir','cashflow'),
    (2,'Pemasukan lebih besar dari pengeluaran dalam 6 bulan terakhir','cashflow'),
    (2,'Pemasukan lebih besar dari pengeluaran dalam 12 bulan terakhir','cashflow'),
    (2,'Rutin mencatat dan mengevaluasi pengeluaran bulanan','cashflow'),
    (3,'Dana darurat 3 bulan pengeluaran tersedia','emergency-fund'),
    (3,'Dana darurat 6 bulan pengeluaran tersedia','emergency-fund'),
    (3,'Dana darurat 12 bulan pengeluaran tersedia','emergency-fund'),
    (4,'BPJS Kesehatan aktif','protection'),
    (4,'Asuransi kesehatan sesuai kebutuhan tersedia','protection'),
    (4,'Asuransi jiwa tersedia jika memiliki tanggungan','protection'),
    (4,'Aset utama sudah memiliki perlindungan','protection'),
    (5,'Tidak memiliki utang konsumtif berbunga tinggi','debt'),
    (5,'Tagihan kartu kredit dibayar penuh setiap bulan','debt'),
    (5,'Rasio kewajiban terhadap pendapatan di bawah 30%','debt'),
    (5,'Punya rencana pelunasan seluruh kewajiban','debt'),
    (6,'Tujuan keuangan sudah dicatat dengan target waktu','goals'),
    (6,'Berinvestasi rutin setiap bulan','investments'),
    (6,'Portofolio investasi sudah terdiversifikasi','investments'),
    (6,'Melakukan evaluasi dan rebalancing berkala','investments'),
    (7,'Target kebutuhan dana pensiun sudah dihitung','retirement'),
    (7,'Menyisihkan dana pensiun secara rutin','retirement'),
    (7,'Proyeksi pensiun mengikuti pendekatan 4%','retirement'),
    (7,'Dokumen waris dan penerima manfaat sudah ditentukan','retirement')
) AS defaults(priority,title,tracker)
ON CONFLICT (workspace_id,priority,title) DO NOTHING;

-- +goose Down

ALTER TABLE pyramid_items
DROP CONSTRAINT IF EXISTS pyramid_items_workspace_priority_title_key;

