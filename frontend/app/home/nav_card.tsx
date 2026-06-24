import type { ReactNode } from 'react';

import { FiArrowRight } from 'react-icons/fi';

import { Link } from 'react-router';

export type NavCardProps = {
	title: string;
	description: string;
	to: string;
	icon?: ReactNode;
};

export function PrimaryActionCard({ title, description, to, icon }: NavCardProps) {
	return (
		<Link to={to} className="group block w-full max-w-lg">
			<div className="bg-base-100 border-primary/20 ring-primary/10 flex min-h-48 items-center gap-5 rounded-3xl border p-4 shadow-xl ring-1 transition-all duration-200 hover:-translate-y-1 hover:shadow-2xl">
				<div className="flex shrink-0 items-center justify-center">
					<div className="bg-primary/10 text-primary rounded-2xl p-4">{icon}</div>
				</div>

				<div className="flex min-w-0 flex-1 flex-col">
					<div className="flex items-center justify-between gap-4">
						<h2 className="text-3xl font-bold">{title}</h2>

						<FiArrowRight size={28} className="shrink-0 transition-transform group-hover:translate-x-1" />
					</div>

					<div className="text-base-content/70 mt-3 text-sm/relaxed">{description}</div>
				</div>
			</div>
		</Link>
	);
}
