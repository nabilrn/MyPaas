import { writable } from 'svelte/store';

export type CreateProjectStep = 'source' | 'configuration' | 'environment' | 'review';

export interface CreateProjectWizardState {
	activeStep: CreateProjectStep;
	sourceComplete: boolean;
	configurationComplete: boolean;
	environmentComplete: boolean;
	reviewReady: boolean;
	busy: boolean;
}

const initialState: CreateProjectWizardState = {
	activeStep: 'source',
	sourceComplete: false,
	configurationComplete: false,
	environmentComplete: false,
	reviewReady: false,
	busy: false
};

export const createProjectWizard = writable<CreateProjectWizardState>(initialState);

export function setCreateProjectStep(activeStep: CreateProjectStep) {
	createProjectWizard.update((state) => ({ ...state, activeStep }));
}

export function updateCreateProjectWizard(state: Omit<CreateProjectWizardState, 'activeStep'>) {
	createProjectWizard.update((current) => ({ ...current, ...state }));
}

export function resetCreateProjectWizard() {
	createProjectWizard.set(initialState);
}
