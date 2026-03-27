import { useEffect, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import { type AuthKeyMeta, type DebugSettings } from '@/spec/setting';

import { settingstoreAPI } from '@/apis/baseapi';

import { DownloadButton } from '@/components/download_button';
import { PageFrame } from '@/components/page_frame';

import { AddEditAuthKeyModal } from '@/settings/authkey_add_edit_modal';
import { AuthKeyTable } from '@/settings/authkey_table';
import { DebugSettingsSection } from '@/settings/debug';
import { ThemeSelector } from '@/settings/theme';

// eslint-disable-next-line no-restricted-exports
export default function SettingsPage() {
	const [authKeys, setAuthKeys] = useState<AuthKeyMeta[]>([]);
	const [debugSettings, setDebugSettings] = useState<DebugSettings | null>(null);
	const [refreshToggle, setRefresh] = useState(false); // helper to force list refresh

	const [isModalOpen, setIsModalOpen] = useState(false);
	const [modalInitial, setModalInitial] = useState<AuthKeyMeta | null>(null); // null = Add

	useEffect(() => {
		let cancelled = false;

		(async () => {
			try {
				const settings = await settingstoreAPI.getSettings();

				if (!cancelled) {
					setAuthKeys(settings.authKeys);
					setDebugSettings(settings.debug);
				}
			} catch (err) {
				console.error('Failed to load settings', err);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [refreshToggle]);

	const refresh = () => {
		setRefresh(p => !p);
	};
	const showAddModal = () => {
		setModalInitial(null);
		setIsModalOpen(true);
	};
	const showEditModal = (meta: AuthKeyMeta) => {
		setModalInitial(meta);
		setIsModalOpen(true);
	};

	const exportSettings = async () => {
		const settings = await settingstoreAPI.getSettings(true);
		return JSON.stringify(settings, null, 2);
	};

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-11/12 items-center px-12 py-2">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">Settings</h1>
					<DownloadButton
						title="Download Settings"
						language="json"
						valueFetcher={exportSettings}
						size={20}
						fileprefix="settings"
						className="btn btn-sm btn-ghost"
						isBinary={false}
					/>
				</div>

				<div
					className="mt-24 flex w-full grow flex-col items-center overflow-y-auto"
					style={{ maxHeight: `calc(100vh - 128px)` }}
				>
					<div className="flex w-11/12 flex-col space-y-4 pb-8 xl:w-2/3">
						<div className="bg-base-100 flex items-center rounded-2xl p-4 shadow-lg">
							<h2 className="mr-8 ml-4 font-semibold">Theme</h2>
							<ThemeSelector />
						</div>

						<div className="bg-base-100 rounded-2xl p-4 shadow-lg">
							<div className="mb-4 flex items-center justify-between">
								<h2 className="mr-8 ml-4 font-semibold">Auth Keys</h2>
								<button className="btn btn-ghost flex items-center rounded-2xl" onClick={showAddModal}>
									<FiPlus className="mr-1" /> Add Key
								</button>
							</div>

							<AuthKeyTable authKeys={authKeys} onEdit={showEditModal} onChanged={refresh} />
						</div>

						<div className="bg-base-100 rounded-2xl p-4 shadow-lg">
							<h2 className="mr-8 mb-4 ml-4 font-semibold">Debug</h2>
							<DebugSettingsSection value={debugSettings} onChanged={setDebugSettings} />
						</div>
					</div>
				</div>

				<AddEditAuthKeyModal
					isOpen={isModalOpen}
					initial={modalInitial}
					existing={authKeys}
					onClose={() => {
						setIsModalOpen(false);
					}}
					onChanged={refresh}
				/>
			</div>
		</PageFrame>
	);
}
